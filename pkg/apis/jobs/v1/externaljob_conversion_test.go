package v1

import (
	"testing"
)

func TestConversion(t *testing.T) {
	// var a, b runtime.Object
	// v1Obj := &ExternalJob{
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       "ExternalJob",
	// 		APIVersion: "jobs.example.org/v1",
	// 	},
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Namespace: "default",
	// 		Name:      "obj-1",
	// 	},
	// }
	// v2Obj := &v2.ExternalJob{}
	// // a, b = v1Obj, v2Obj
	//
	// _, err := SchemeBuilder.Build()
	// if err != nil {
	// 	t.Errorf("error building the scheme")
	// }

	// err = convertObject(myscheme, a, b)
	// if err != nil {
	// 	t.Errorf("failed to convert: %v ", err)
	// }
	// t.Logf("converted object v2obj %+v", v2Obj.ObjectMeta)
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

// func enumerateVersions(t *testing.T) {
// 	t.Logf("enumerating all the versions")
//
// 	myscheme, err := SchemeBuilder.Build()
// 	if err != nil {
// 		t.Errorf("error building the scheme")
// 		return
// 	}
//
// 	for k, v := range myscheme.AllKnownTypes() {
// 		t.Logf("k %v v %v", k, v)
// 	}
//
// 	hub, err := getHub(myscheme, &ExternalJob{})
// 	if err != nil {
// 		t.Errorf("error retrieving hub %v", err)
// 	}
// 	t.Logf("found a hub %T", hub)
// }
