//go:build darwin
// +build darwin

package metal

import (
	"reflect"
	"unsafe"
)

// sizeof returns the size in bytes of the generic type T.
func sizeof[T any]() int {
	var t T
	return int(unsafe.Sizeof(t))
}

// toSlice transforms a block of memory into a go slice. It wraps the memory inside a slice header
// and sets the len/cap to the number of elements. This is unsafe behavior and can lead to data
// corruption.
func toSlice[T any](data unsafe.Pointer, numElems int) []T {
	// Create a slice header with the generic type for a slice that has no backing array.
	var s []T

	// Cast the slice header into a reflect.SliceHeader so we can actually access the slice's
	// internals and set our own values. In effect, this wraps a go slice around our data so we can
	// access it natively.
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&s))

	// Set our data in the slice internals.
	hdr.Data = uintptr(data)
	hdr.Len = numElems
	hdr.Cap = numElems

	return s
}
