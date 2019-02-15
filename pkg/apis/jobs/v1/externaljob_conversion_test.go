package v1

import (
	"fmt"
	"testing"

	"github.com/droot/crd-conversion-example/pkg/apis/jobs/v2"
	"github.com/droot/crd-conversion-example/pkg/conversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func TestConversion(t *testing.T) {
	var a, b runtime.Object
	v1Obj := &ExternalJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ExternalJob",
			APIVersion: "jobs.example.org/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "obj-1",
		},
	}
	v2Obj := &v2.ExternalJob{}
	a, b = v1Obj, v2Obj

	myscheme, err := SchemeBuilder.Build()
	if err != nil {
		t.Errorf("error building the scheme")
	}

	err = convert(myscheme, a, b)
	if err != nil {
		t.Errorf("failed to convert: %v ", err)
	}
	t.Logf("converted object v2obj %+v", v2Obj.ObjectMeta)
	// if con, ok := a.(conversion.Convertable); ok {
	// 	err := con.ConvertTo(b)
	// 	if err != nil {
	// 		t.Errorf("failed to convert v1 to v2 obj: %v", err)
	// 	} else {
	// 		t.Logf("converted object v2obj %+v", v2Obj.ObjectMeta)
	// 	}
	// } else {
	// 	t.Logf("object is not convertable")
	// }
	// enumerateVersions(t)
}

func enumerateVersions(t *testing.T) {
	t.Logf("enumerating all the versions")

	myscheme, err := SchemeBuilder.Build()
	if err != nil {
		t.Errorf("error building the scheme")
		return
	}

	for k, v := range myscheme.AllKnownTypes() {
		t.Logf("k %v v %v", k, v)
	}

	hub, err := getHub(myscheme, &ExternalJob{})
	if err != nil {
		t.Errorf("error retrieving hub %v", err)
	}
	t.Logf("found a hub %T", hub)
}

func isHub(obj runtime.Object) bool {
	_, yes := obj.(conversion.Hub)
	return yes
}

func isConvertable(obj runtime.Object) bool {
	_, yes := obj.(conversion.Convertable)
	return yes
}

func convert(myscheme *runtime.Scheme, a, b runtime.Object) error {
	// check if a and b are of same type, then just do deepCopy ?

	// TODO(droot): figure out a less verbose version of this check
	if a.GetObjectKind().GroupVersionKind().String() == b.GetObjectKind().GroupVersionKind().String() {
		// maybe, we should just use deepcopy here ?
		return fmt.Errorf("conversion is not allowed between same type %T", a)
	}

	aIsHub, bIsHub := isHub(a), isHub(b)
	aIsConvertable, bIsConvertable := isConvertable(a), isConvertable(b)

	if aIsHub {
		if bIsConvertable {
			return b.(conversion.Convertable).ConvertFrom(a)
		} else {
			// this is error case ?
			return fmt.Errorf("%T is not convertable to", a)
		}
	}

	if bIsHub {
		if aIsConvertable {
			return a.(conversion.Convertable).ConvertTo(b)
		} else {
			return fmt.Errorf("%T is not convertable", a)
		}
	}

	// neigher a nor b are Hub, means both of them are spoke, so lets get the hub
	// version type.
	hub, err := getHub(myscheme, a)
	if err != nil {
		return err
	}
	// shall we get Hub for b type as well and ensure hubs are same ?

	// a and b needs to be convertable for it to work
	if !aIsConvertable || !bIsConvertable {
		return fmt.Errorf("%T and %T needs to be both convertable", a, b)
	}

	err = a.(conversion.Convertable).ConvertTo(hub)
	if err != nil {
		return fmt.Errorf("%T failed to convert to hub version %T : %v", a, hub, err)
	}

	err = b.(conversion.Convertable).ConvertFrom(hub)
	if err != nil {
		return fmt.Errorf("%T failed to convert from hub version %T : %v", b, hub, err)
	}

	return nil
}

func getHub(myscheme *runtime.Scheme, obj runtime.Object) (runtime.Object, error) {
	gvks, _, err := myscheme.ObjectKinds(obj)
	if err != nil {
		return nil, fmt.Errorf("error retriving object kinds for given object : %v", err)
	}

	var hub runtime.Object
	hubFoundAlready := false
	for _, gvk := range gvks {
		o, _ := myscheme.New(gvk)
		if _, IsHub := o.(conversion.Hub); IsHub {
			if hubFoundAlready {
				// multiple hub found, error case
				return nil, fmt.Errorf("multiple hub version defined")
			}
			hubFoundAlready = true
			hub = o
		}
	}
	return hub, nil
}
