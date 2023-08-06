//go:build darwin
// +build darwin

package metal

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"errors"
	"fmt"
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

// metalErrToError wraps the metal error metalErr inside wrap.
func metalErrToError(metalErr *C.char, wrap string) error {
	switch {
	case metalErr == nil || C.strlen(metalErr) == 0:
		if wrap == "" {
			// We have neither a metal error nor any wrapping. Return nil.
			return nil
		}

		// We have wrapping but we don't have a metal error. Return just the wrapping.
		return errors.New(wrap)

	default:
		mErr := errors.New(C.GoString(metalErr))

		if wrap == "" {
			// We have a metal error but we don't have any wrapping. Return just the metal error.
			return mErr
		}

		// We have both a metal error and wrapping. Return both of them formatted together.
		return fmt.Errorf("%s: %w", wrap, mErr)
	}
}

// cgoString and cgoFree are wrappers around cgo functions to enable using cgo in test files.
func cgoString(s string) *C.char { return C.CString(s) }
func cgoFree(s *C.char)          { C.free(unsafe.Pointer(s)) }
