//go:build darwin
// +build darwin

package metal

// frameworks not included:
// Cocoa

/*
#cgo LDFLAGS: -framework Metal -framework CoreGraphics -framework Foundation
#include "metal.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

func init() {
	// Initialize the device that will be used to run the computations.
	C.metal_init()
}

// A BufferId references a specific metal buffer created with NewBuffer.
type BufferId int

// Valid checks whether or not the buffer Id is valid and can be used to run a computational process
// on the GPU.
func (id BufferId) Valid() bool {
	return id > 0
}

type BufferType interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// NewBuffer1D allocates a 1-dimensional block of memory that is accessible to both the CPU and GPU.
// It returns a unique Id for the buffer and a slice that wraps the new memory and has a length and
// capacity equal to length.
//
// The Id is used to reference the buffer as an argument for the metal function.
//
// Only the contents of the slice should be modified. Its length and capacity and the pointer to its
// underlying array should not be altered.
func NewBuffer1D[T BufferType](length int) (BufferId, []T, error) {
	return newBuffer[T](length)
}

// NewBuffer2D allocates a 2-dimensional block of memory that is accessible to both the CPU and GPU.
// It returns a unique Id for the buffer and a slice that wraps the new memory and has a length and
// capacity equal to length. Each element in the slice is another slice with a length equal to
// width.
//
// The Id is used to reference the buffer as an argument for the metal function.
//
// Only the contents of the slices should be modified. Their lengths and capacities and the pointers
// to their underlying arrays should not be altered.
func NewBuffer2D[T BufferType](length, width int) (BufferId, [][]T, error) {
	bufferId, b1, err := newBuffer[T](length, width)
	if err != nil {
		return 0, nil, err
	}

	b2 := fold(b1, length)

	return bufferId, b2, nil
}

// NewBuffer3D allocates a 3-dimensional block of memory that is accessible to both the CPU and GPU.
// It returns a unique Id for the buffer and a slice that wraps the new memory and has a length and
// capacity equal to length. Each element in the slice is another slice with a length equal to
// width, and each of their elements is in turn another slice with a length equal to height.
//
// The Id is used to reference the buffer as an argument for the metal function.
//
// Only the contents of the slices should be modified. Their lengths and capacities and the pointers
// to their underlying arrays should not be altered.
func NewBuffer3D[T BufferType](length, width, height int) (BufferId, [][][]T, error) {
	bufferId, b1, err := newBuffer[T](length, width, height)
	if err != nil {
		return 0, nil, err
	}

	b2 := fold(b1, length*width)
	b3 := fold(b2, length)

	return bufferId, b3, nil
}

func newBuffer[T BufferType](dimLens ...int) (BufferId, []T, error) {
	if len(dimLens) == 0 {
		return 0, nil, errors.New("Missing dimension(s)")
	}
	for _, dimLen := range dimLens {
		if dimLen < 1 {
			return 0, nil, errors.New("Invalid number of elements")
		}
	}

	numElems := 1
	for _, dimLen := range dimLens {
		numElems *= dimLen
	}
	numBytes := sizeof[T]() * numElems

	err := C.CString("")
	defer C.free(unsafe.Pointer(err))

	// Allocate memory for the new buffer.
	bufferId := C.buffer_new(C.int(numBytes), &err)
	if int(bufferId) == 0 {
		return 0, nil, metalErrToError(err, "Unable to create buffer")
	}

	// Retrieve a pointer to the beginning of the new memory using the buffer's Id.
	newBuffer := C.buffer_retrieve(bufferId, &err)
	if newBuffer == nil {
		return 0, nil, metalErrToError(err, "Unable to retrieve buffer")
	}

	return BufferId(bufferId), toSlice[T](newBuffer, numElems), nil
}