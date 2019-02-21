package conversion

import "k8s.io/apimachinery/pkg/runtime"

// A versioned type is convertable if it can be converted to/from a hub type.
type Convertable interface {
	runtime.Object
	ConvertTo(dst Hub) error
	ConvertFrom(src Hub) error
}

// Hub defines capability to indicate whether a versioned type is a Hub or not.
// Default conversion handler will use this interface to implement spoke to
// spoke conversion.
type Hub interface {
	runtime.Object
	Hub()
}
