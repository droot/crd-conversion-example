package conversion

import "k8s.io/apimachinery/pkg/runtime"

// Convertable defines the roundtrip capability between Hub/Spoke.
type Convertable interface {
	ConvertTo(dst runtime.Object) error
	ConvertFrom(src runtime.Object) error
}

// Hub defines capability to indicate whether a GVK is a Hub or not ?
type Hub interface {
	Hub()
}
