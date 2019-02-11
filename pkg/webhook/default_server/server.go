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
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/droot/crd-conversion-example/pkg/apis/jobs/v1"
	"github.com/droot/crd-conversion-example/pkg/apis/jobs/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"

	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

var (
	log        = logf.Log.WithName("default_server")
	builderMap = map[string]*builder.WebhookBuilder{}
	// HandlerMap contains all admission webhook handlers.
	HandlerMap = map[string][]admission.Handler{}
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
	for k, builder := range builderMap {
		handlers, ok := HandlerMap[k]
		if !ok {
			log.V(1).Info(fmt.Sprintf("can't find handlers for builder: %v", k))
			handlers = []admission.Handler{}
		}
		wh, err := builder.
			Handlers(handlers...).
			WithManager(mgr).
			Build()
		if err != nil {
			return err
		}
		webhooks = append(webhooks, wh)
	}

	svr.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		log.Info("got a convert request")

		// body, err := ioutil.ReadAll(r.Body)
		// if err != nil {
		// 	log.Error(err, "error reading the request body")
		// }
		convertReview := &apix.ConversionReview{}

		// d := newDecoder(mgr.GetScheme())
		err := json.NewDecoder(r.Body).Decode(convertReview)
		if err != nil {
			log.Error(err, "error decoding conversion request")
			// TODO(droot): define helper for returning conversion error
			// response
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		convertReview.Response = doConversion(mgr.GetScheme(), convertReview.Request)
		convertReview.Response.UID = convertReview.Request.UID

		err = json.NewEncoder(w).Encode(convertReview)
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

// doConversion converts the requested object given the conversion function and returns a conversion response.
// failures will be reported as Reason in the conversion response.
func doConversion(scheme *runtime.Scheme, convertRequest *apix.ConversionRequest) *apix.ConversionResponse {
	var convertedObjects []runtime.RawExtension

	var conversionCodecs = serializer.NewCodecFactory(scheme)
	for _, obj := range convertRequest.Objects {
		log.Info("decoding object", "object", obj)
		a, b, err := conversionCodecs.UniversalDeserializer().Decode(obj.Raw, nil, nil)
		if err != nil {
			log.Error(err, "error decoding to v1 obj")
		}
		log.Info("decoding incoming obj", "a", a, "b", b, "a-type", fmt.Sprintf("%T", a))
		switch convertRequest.DesiredAPIVersion {
		case "jobs.example.org/v2":
			v1Obj := &v1.ExternalJob{}
			// if err := d.Decode(obj.Raw, v1Obj); err != nil {
			// 	log.Error(err, "error decoding to v1 obj")
			// }
			// log.Info("successfully decoded to obj v1", "object-v1", v1Obj)
			// do the conversion here
			convertedObjects = append(convertedObjects, runtime.RawExtension{Object: v1Obj})
		case "jobs.example.org/v1":
			v2Obj := &v2.ExternalJob{}
			// if err := d.Decode(obj.Raw, v2Obj); err != nil {
			// 	log.Error(err, "error decoding to v1 obj")
			// }
			log.Info("successfully decoded to obj v2", "object-v2", v2Obj)
			convertedObjects = append(convertedObjects, runtime.RawExtension{Object: v2Obj})
		default:
			return conversionResponseFailureWithMessagef("unknown desired version")
		}
		// cr := unstructured.Unstructured{}
		// if err := cr.UnmarshalJSON(obj.Raw); err != nil {
		// 	log.Error(err, "failed to unmarshal object", "object", obj.Raw)
		// 	return conversionResponseFailureWithMessagef("failed to unmarshall object (%v) with error: %v", string(obj.Raw), err)
		// }
		// convertedCR, status := convert(&cr, convertRequest.DesiredAPIVersion)
		// if status.Status != metav1.StatusSuccess {
		// 	klog.Error(status.String())
		// 	return &v1beta1.ConversionResponse{
		// 		Result: status,
		// 	}
		// }
		// convertedCR.SetAPIVersion(convertRequest.DesiredAPIVersion)
		// convertedObjects = append(convertedObjects, runtime.RawExtension{Object: convertedCR})
	}
	return &apix.ConversionResponse{
		ConvertedObjects: convertedObjects,
		Result:           statusSucceed(),
	}
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
