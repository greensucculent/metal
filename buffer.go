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

type BufferType interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// NewBuffer allocates a block of memory that is accessible to both the CPU and GPU. It returns a
// unique Id for the buffer and a slice that wraps the new memory and has a len and cap equal to
// numElems.
//
// The Id is used to reference the buffer as an argument for the metal function.
//
// Only the contents of the slice should be modified. Its length and capacity and the block of
// memory that it points to should not be altered. The slice's length and capacity are equal to
// numElems, and its underlying memory has (numElems * sizeof(T)) bytes.
func NewBuffer[T any](numElems int) (BufferId, []T, error) {
	if numElems <= 0 {
		return 0, nil, errors.New("Invalid number of elements")
	}

	elemSize := sizeof[T]()
	numBytes := elemSize * numElems

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

// Valid checks whether or not the buffer Id is valid and can be used to run a computational process
// on the GPU.
func (id BufferId) Valid() bool {
	return id > 0
}
