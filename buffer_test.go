//go:build darwin
// +build darwin

package metal

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_NewBuffer tests that NewBuffer creates a new metal buffer with the expected underlying type
// and number of elements.
func Test_NewBuffer(t *testing.T) {

	// Invalid configuration (no elements).
	bufferId, buffer, err := NewBuffer[int32](0)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer)

	// Invalid configuration (negative elements).
	bufferId, buffer, err = NewBuffer[int32](-1)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer)

	// Test the primitive types that satisfy the BufferType constraint.
	testNewBuffer(t, func(i int) byte { return byte(i) })
	testNewBuffer(t, func(i int) rune { return rune(i) })
	testNewBuffer(t, func(i int) uint8 { return uint8(i) })
	testNewBuffer(t, func(i int) uint16 { return uint16(i) })
	testNewBuffer(t, func(i int) uint32 { return uint32(i) })
	testNewBuffer(t, func(i int) uint64 { return uint64(i) })
	testNewBuffer(t, func(i int) int8 { return int8(i) })
	testNewBuffer(t, func(i int) int16 { return int16(i) })
	testNewBuffer(t, func(i int) int32 { return int32(i) })
	testNewBuffer(t, func(i int) int64 { return int64(i) })
	testNewBuffer(t, func(i int) float32 { return float32(i) })
	testNewBuffer(t, func(i int) float64 { return float64(i) })

	// Test custom types that satisfy the BufferType constraint.
	type MyByte byte
	testNewBuffer(t, func(i int) MyByte { return MyByte(i) })
	type MyRune rune
	testNewBuffer(t, func(i int) MyRune { return MyRune(i) })
	type MyUint8 uint8
	testNewBuffer(t, func(i int) MyUint8 { return MyUint8(i) })
	type MyUint16 uint16
	testNewBuffer(t, func(i int) MyUint16 { return MyUint16(i) })
	type MyUint32 uint32
	testNewBuffer(t, func(i int) MyUint32 { return MyUint32(i) })
	type MyUint64 uint64
	testNewBuffer(t, func(i int) MyUint64 { return MyUint64(i) })
	type MyInt8 int8
	testNewBuffer(t, func(i int) MyInt8 { return MyInt8(i) })
	type MyInt16 int16
	testNewBuffer(t, func(i int) MyInt16 { return MyInt16(i) })
	type MyInt32 int32
	testNewBuffer(t, func(i int) MyInt32 { return MyInt32(i) })
	type MyInt64 int64
	testNewBuffer(t, func(i int) MyInt64 { return MyInt64(i) })
	type MyFloat32 float32
	testNewBuffer(t, func(i int) MyFloat32 { return MyFloat32(i) })
	type MyFloat64 float64
	testNewBuffer(t, func(i int) MyFloat64 { return MyFloat64(i) })
}

// testNewBuffer is a helper to test buffer creation for a variety of types.
func testNewBuffer[T BufferType](t *testing.T, converter func(int) T) {
	var a T
	t.Run(fmt.Sprintf("%T", a), func(t *testing.T) {

		bufferId, buffer, err := NewBuffer[T](10)
		require.Nil(t, err, "Unable to create metal buffer: %s", err)
		require.True(t, validId(bufferId))
		require.Len(t, buffer, 10)
		require.Equal(t, len(buffer), cap(buffer))

		// Test that every item in the buffer has its zero value.
		for i := range buffer {
			require.True(t, reflect.ValueOf(buffer[i]).IsZero())
		}

		// Test that we can write to every item in the buffer.
		require.NotPanics(t, func() {
			for i := range buffer {
				buffer[i] = converter(i)
			}
		})

		// Test that every item retained its value.
		for i := range buffer {
			require.Equal(t, converter(i), buffer[i])
		}
	})
}

// Test_Valid tests that BufferId's Valid method correctly identifies a valid BufferId.
func Test_Valid(t *testing.T) {
	// A valid BufferId has a positive value. Let's run through a bunch of numbers and that Valid
	// always report the correct status.
	for i := -100_00; i <= 100_000; i++ {
		bufferId := BufferId(i)

		if i > 0 {
			require.True(t, bufferId.Valid())
		} else {
			require.False(t, bufferId.Valid())
		}
	}
}

// Test_ThreadSafe tests that NewBuffer can handle multiple parallel invocations and still return
// the correct Id.
func Test_ThreadSafe(t *testing.T) {
	// We're going to use a wait group to block each goroutine after it's prepared until they're all
	// ready to fire.
	numIter := 100
	var wg sync.WaitGroup
	wg.Add(numIter)

	dataCh := make(chan BufferId)

	// Prepare one goroutine to create a new buffer for each iteration.
	for i := 0; i < numIter; i++ {
		// Calculate the length for this buffer.
		length := i + 1

		// Spin up a new goroutine. This will wait until all goroutines are ready to fire, then
		// create a new metal buffer and send its Id back to the main thread.
		go func() {
			wg.Wait()

			bufferId, _, err := NewBuffer[int32](length)
			require.Nil(t, err, "Unable to create metal buffer: %s", err)

			dataCh <- bufferId
		}()

		// Mark that this goroutine is ready.
		wg.Done()
	}

	// Check that each buffer's Id is unique.
	idMap := make(map[BufferId]struct{})
	for i := 0; i < numIter; i++ {
		bufferId := <-dataCh

		_, ok := idMap[bufferId]
		require.False(t, ok)
		idMap[bufferId] = struct{}{}

		addId()
	}
}
