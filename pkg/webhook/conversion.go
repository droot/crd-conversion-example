package webhook

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/droot/crd-conversion-example/pkg/conversion"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"

	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log = logf.Log.WithName("conversion_webhook")
)

type ConversionHandler struct {
	Scheme *runtime.Scheme

	serializer runtime.Serializer
	// decoder
	// TODO(droot): make scheme and decoder injectable

	once sync.Once
}

func (cb *ConversionHandler) setDefaults() {
	cb.once.Do(func() {
		if cb.Scheme == nil {
			cb.Scheme = runtime.NewScheme()
		}
		cb.serializer = json.NewSerializer(json.DefaultMetaFactory, cb.Scheme, cb.Scheme, false)
		// conversionCodecs := serializer.NewCodecFactory(cb.Scheme)
	})
}

// ensure ConversionHandler implements http.Handler
var _ http.Handler = &ConversionHandler{}

func (cb *ConversionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cb.setDefaults()
	log.Info("got a convert request")

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	convertReview := apix.ConversionReview{}

	// serializer := json.NewSerializer(json.DefaultMetaFactory, cb.Scheme, cb.Scheme, false)
	_, _, err := cb.serializer.Decode(body, nil, &convertReview)
	if err != nil {
		log.Error(err, "error decoding conversion request")
		// TODO(droot): define helper for returning conversion error response
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	convertReview.Response = cb.handleConvertRequest(convertReview.Request)
	convertReview.Response.UID = convertReview.Request.UID

	err = cb.serializer.Encode(&convertReview, w)
	if err != nil {
		log.Error(err, "error encoding conversion request")
		// TODO(droot): define helper for returning conversion error
		// response
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// handles a version conversion request.
func (cb *ConversionHandler) handleConvertRequest(req *apix.ConversionRequest) *apix.ConversionResponse {
	var convertedObjects []runtime.RawExtension

	for _, obj := range req.Objects {
		// src, gvk, err := cb.conversionCodecs.UniversalDeserializer().Decode(obj.Raw, nil, nil)
		src, gvk, err := cb.serializer.Decode(obj.Raw, nil, nil)
		if err != nil {
			log.Error(err, "error decoding src object")
		}
		log.Info("decoding incoming obj", "src", src, "gvk", gvk, "src type", fmt.Sprintf("%T", src))

		dst, err := getTargetObject(cb.Scheme, req.DesiredAPIVersion, gvk.Kind)
		if err != nil {
			log.Error(err, "error getting destination object")
			return conversionResponseFailureWithMessagef("error converting object")
		}
		err = cb.convertObject(src, dst)
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

func (cb *ConversionHandler) convertObject(src, dst runtime.Object) error {
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
	hub, err := getHub(cb.Scheme, src)
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
