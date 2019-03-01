/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package defaultserver

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/droot/crd-conversion-example/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	log = logf.Log.WithName("default_server")
	// builderMap = map[string]*builder.WebhookBuilder{}
	// HandlerMap contains all admission webhook handlers.
	// HandlerMap = map[string][]admission.Handler{}
)

// Add adds itself to the manager
func Add(mgr manager.Manager) error {
	ns := os.Getenv("POD_NAMESPACE")
	if len(ns) == 0 {
		ns = "default"
	}
	secretName := os.Getenv("SECRET_NAME")
	if len(secretName) == 0 {
		secretName = "webhook-server-secret"
	}

	disableInstaller := true

	// +kubebuilder:webhook:port=9876,cert-dir=/tmp/cert
	// +kubebuilder:webhook:service=crd-cc-system:crd-cc-webhook-service,selector=control-plane:controller-manager
	// +kubebuilder:webhook:secret=system:webhook-server-secret
	// +kubebuilder:webhook:validating-webhook-config-name=test-mutating-webhook-cfg,validating-webhook-config-name=test-validating-webhook-cfg
	svr, err := webhook.NewServer("foo-admission-server", mgr, webhook.ServerOptions{
		// TODO(user): change the configuration of ServerOptions based on your need.
		Port:                          9876,
		CertDir:                       "/tmp/cert",
		DisableWebhookConfigInstaller: &disableInstaller,
		BootstrapOptions: &webhook.BootstrapOptions{
			Secret: &types.NamespacedName{
				Namespace: ns,
				Name:      secretName,
			},

			Service: &webhook.Service{
				Namespace: ns,
				Name:      "crd-cc-webhook-service",
				// Selectors should select the pods that runs this webhook server.
				Selectors: map[string]string{
					"control-plane": "controller-manager",
				},
			},
		},
	})
	if err != nil {
		return err
	}

	var webhooks []webhook.Webhook
	// for k, builder := range builderMap {
	// 	handlers, ok := HandlerMap[k]
	// 	if !ok {
	// 		log.V(1).Info(fmt.Sprintf("can't find handlers for builder: %v", k))
	// 		handlers = []admission.Handler{}
	// 	}
	// 	wh, err := builder.
	// 		Handlers(handlers...).
	// 		WithManager(mgr).
	// 		Build()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	webhooks = append(webhooks, wh)
	// }

	svr.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		log.Info("got a convert request")

		var body []byte
		if r.Body != nil {
			if data, err := ioutil.ReadAll(r.Body); err == nil {
				body = data
			}
		}
		convertReview := apix.ConversionReview{}

		serializer := json.NewSerializer(json.DefaultMetaFactory, mgr.GetScheme(), mgr.GetScheme(), false)
		_, _, err := serializer.Decode(body, nil, &convertReview)
		if err != nil {
			log.Error(err, "error decoding conversion request")
			// TODO(droot): define helper for returning conversion error response
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		convertReview.Response = handleConvertRequest(serializer, mgr.GetScheme(), convertReview.Request)
		convertReview.Response.UID = convertReview.Request.UID

		err = serializer.Encode(&convertReview, w)
		if err != nil {
			log.Error(err, "error encoding conversion request")
			// TODO(droot): define helper for returning conversion error
			// response
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	return svr.Register(webhooks...)
}

// handles a version conversion request.
func handleConvertRequest(ser *json.Serializer, scheme *runtime.Scheme, req *apix.ConversionRequest) *apix.ConversionResponse {
	var convertedObjects []runtime.RawExtension

	var conversionCodecs = serializer.NewCodecFactory(scheme)
	for _, obj := range req.Objects {
		src, gvk, err := conversionCodecs.UniversalDeserializer().Decode(obj.Raw, nil, nil)
		if err != nil {
			log.Error(err, "error decoding src object")
		}
		log.Info("decoding incoming obj", "src", src, "gvk", gvk, "src type", fmt.Sprintf("%T", src))

		dst, err := getTargetObject(scheme, req.DesiredAPIVersion, gvk.Kind)
		if err != nil {
			log.Error(err, "error getting destination object")
			return conversionResponseFailureWithMessagef("error converting object")
		}
		err = convertObject(scheme, src, dst)
		if err != nil {
			log.Error(err, "error converting object")
			return conversionResponseFailureWithMessagef("error converting object")
		}
		convertedObjects = append(convertedObjects, runtime.RawExtension{Object: dst})
	}
	return &apix.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result:           statusSucceed(),
	}
}

func convertObject(scheme *runtime.Scheme, src, dst runtime.Object) error {
	// TODO(droot): figure out a less verbose version of this check
	if src.GetObjectKind().GroupVersionKind().String() == dst.GetObjectKind().GroupVersionKind().String() {
		return fmt.Errorf("conversion is not allowed between same type %T", src)
	}

	srcIsHub, dstIsHub := isHub(src), isHub(dst)
	srcIsConvertable, dstIsConvertable := isConvertable(src), isConvertable(dst)

	if srcIsHub {
		if dstIsConvertable {
			return dst.(conversion.Convertable).ConvertFrom(src.(conversion.Hub))
		} else {
			// this is error case, this can be flagged at setup time ?
			return fmt.Errorf("%T is not convertable to", src)
		}
	}

	if dstIsHub {
		if srcIsConvertable {
			return src.(conversion.Convertable).ConvertTo(dst.(conversion.Hub))
		} else {
			// this is error case.
			return fmt.Errorf("%T is not convertable", src)
		}
	}

	// neigher src nor dst are Hub, means both of them are spoke, so lets get the hub
	// version type.
	hub, err := getHub(scheme, src)
	if err != nil {
		return err
	}
	// shall we get Hub for dst type as well and ensure hubs are same ?

	// src and dst needs to be convertable for it to work
	if !srcIsConvertable || !dstIsConvertable {
		return fmt.Errorf("%T and %T needs to be both convertable", src, dst)
	}

	err = src.(conversion.Convertable).ConvertTo(hub)
	if err != nil {
		return fmt.Errorf("%T failed to convert to hub version %T : %v", src, hub, err)
	}

	err = dst.(conversion.Convertable).ConvertFrom(hub)
	if err != nil {
		return fmt.Errorf("%T failed to convert from hub version %T : %v", dst, hub, err)
	}

	return nil
}

func getHub(scheme *runtime.Scheme, obj runtime.Object) (conversion.Hub, error) {
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		return nil, fmt.Errorf("error retriving object kinds for given object : %v", err)
	}

	var hub conversion.Hub
	hubFoundAlready := false
	var isHub bool
	for _, gvk := range gvks {
		o, _ := scheme.New(gvk)
		if hub, isHub = o.(conversion.Hub); isHub {
			if hubFoundAlready {
				// multiple hub found, error case
				return nil, fmt.Errorf("multiple hub version defined")
			}
			hubFoundAlready = true
		}
	}
	return hub, nil
}

func getTargetObject(scheme *runtime.Scheme, apiVersion, kind string) (runtime.Object, error) {
	gvk := schema.FromAPIVersionAndKind(apiVersion, kind)

	obj, err := scheme.New(gvk)
	if err != nil {
		return obj, err
	}

	t, err := meta.TypeAccessor(obj)
	if err != nil {
		return obj, err
	}

	t.SetAPIVersion(apiVersion)
	t.SetKind(kind)

	return obj, nil
}

// conversionResponseFailureWithMessagef is a helper function to create an AdmissionResponse
// with a formatted embedded error message.
func conversionResponseFailureWithMessagef(msg string, params ...interface{}) *apix.ConversionResponse {
	return &apix.ConversionResponse{
		Result: metav1.Status{
			Message: fmt.Sprintf(msg, params...),
			Status:  metav1.StatusFailure,
		},
	}

}

func statusErrorWithMessage(msg string, params ...interface{}) metav1.Status {
	return metav1.Status{
		Message: fmt.Sprintf(msg, params...),
		Status:  metav1.StatusFailure,
	}
}

func statusSucceed() metav1.Status {
	return metav1.Status{
		Status: metav1.StatusSuccess,
	}
}

func isHub(obj runtime.Object) bool {
	_, yes := obj.(conversion.Hub)
	return yes
}

func isConvertable(obj runtime.Object) bool {
	_, yes := obj.(conversion.Convertable)
	return yes
}
