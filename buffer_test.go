//go:build darwin
// +build darwin

package metal

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test_BufferId_Valid tests that BufferId's Valid method correctly identifies a valid buffer Id.
func Test_BufferId_Valid(t *testing.T) {
	// A valid buffer Id has a positive value. Let's run through a bunch of numbers and test that
	// Valid always report the correct status.
	for i := -100_00; i <= 100_000; i++ {
		bufferId := BufferId(i)

		if i > 0 {
			require.True(t, bufferId.Valid())
		} else {
			require.False(t, bufferId.Valid())
		}
	}
}

// Test_NewBuffer_invalid tests that each of the NewBuffer implementations handles invalid
// arguments correctly.
func Test_NewBuffer_invalid(t *testing.T) {
	// 1D: no length
	bufferId, buffer1D, err := NewBuffer1D[int32](0)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer1D)

	// 1D: negative length
	bufferId, buffer1D, err = NewBuffer1D[int32](-1)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer1D)

	// 2D: no length.
	bufferId, buffer2D, err := NewBuffer2D[int32](0, 10)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer2D)

	// 2D: no width.
	bufferId, buffer2D, err = NewBuffer2D[int32](10, 0)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer2D)

	// 2D: negative length.
	bufferId, buffer2D, err = NewBuffer2D[int32](-1, 10)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer2D)

	// 2D: negative width.
	bufferId, buffer2D, err = NewBuffer2D[int32](10, -1)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer2D)

	// 3D: no length.
	bufferId, buffer3D, err := NewBuffer3D[int32](0, 10, 10)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer3D)

	// 3D: no width.
	bufferId, buffer3D, err = NewBuffer3D[int32](10, 0, 10)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer3D)

	// 3D: no height.
	bufferId, buffer3D, err = NewBuffer3D[int32](10, 10, 0)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer3D)

	// 3D: negative length.
	bufferId, buffer3D, err = NewBuffer3D[int32](-1, 10, 10)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer3D)

	// 3D: negative width.
	bufferId, buffer3D, err = NewBuffer3D[int32](10, -1, 10)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer3D)

	// 3D: negative height.
	bufferId, buffer3D, err = NewBuffer3D[int32](10, 10, -1)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer3D)
}

// Test_NewBuffer tests that each of the NewBuffer implementations creates a new metal buffer with
// the expected underlying type and data shape.
func Test_NewBuffer(t *testing.T) {
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

	// Test a 1-dimensional buffer.
	t.Run(fmt.Sprintf("%T_1D", a), func(t *testing.T) {

		length := rand.Intn(20) + 1

		bufferId, buffer, err := NewBuffer1D[T](length)
		require.Nil(t, err, "Unable to create metal buffer: %s", err)
		require.True(t, validId(bufferId))
		require.Len(t, buffer, length)
		require.Equal(t, cap(buffer), length)

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

	// Test a 2-dimensional buffer.
	t.Run(fmt.Sprintf("%T_2D", a), func(t *testing.T) {

		length := rand.Intn(20) + 1
		width := rand.Intn(20) + 1

		bufferId, buffer, err := NewBuffer2D[T](length, width)
		require.Nil(t, err, "Unable to create metal buffer: %s", err)
		require.True(t, validId(bufferId))
		require.Equal(t, length, len(buffer))
		require.Equal(t, length, cap(buffer))
		for _, y := range buffer {
			require.Equal(t, width, len(y))
			require.Equal(t, width, cap(y))
		}

		// Test that every item in the buffer has its zero value.
		for i := range buffer {
			for j := range buffer[i] {
				require.True(t, reflect.ValueOf(buffer[i][j]).IsZero())
			}
		}

		// Test that we can write to every item in the buffer.
		require.NotPanics(t, func() {
			for i := range buffer {
				for j := range buffer[i] {
					buffer[i][j] = converter(i * j)
				}
			}
		})

		// Test that every item retained its value.
		for i := range buffer {
			for j := range buffer[i] {
				require.Equal(t, converter(i*j), buffer[i][j])
			}
		}
	})

	// Test a 3-dimensional buffer.
	t.Run(fmt.Sprintf("%T_3D", a), func(t *testing.T) {

		length := rand.Intn(20) + 1
		width := rand.Intn(20) + 1
		height := rand.Intn(20) + 1

		bufferId, buffer, err := NewBuffer3D[T](length, width, height)
		require.Nil(t, err, "Unable to create metal buffer: %s", err)
		require.True(t, validId(bufferId))
		require.Equal(t, length, len(buffer))
		require.Equal(t, length, cap(buffer))
		for _, y := range buffer {
			require.Equal(t, width, len(y))
			require.Equal(t, width, cap(y))
			for _, z := range y {
				require.Equal(t, height, len(z))
				require.Equal(t, height, cap(z))
			}
		}

		// Test that every item in the buffer has its zero value.
		for i := range buffer {
			for j := range buffer[i] {
				for k := range buffer[i][j] {
					require.True(t, reflect.ValueOf(buffer[i][j][k]).IsZero())
				}
			}
		}

		// Test that we can write to every item in the buffer.
		require.NotPanics(t, func() {
			for i := range buffer {
				for j := range buffer[i] {
					for k := range buffer[i][j] {
						buffer[i][j][k] = converter(i * j * k)
					}
				}
			}
		})

		// Test that every item retained its value.
		for i := range buffer {
			for j := range buffer[i] {
				for k := range buffer[i][j] {
					require.Equal(t, converter(i*j*k), buffer[i][j][k])
				}
			}
		}
	})
}

// Test_NewBuffer_threadSafe tests that each of the NewBuffer implementations can handle multiple
// parallel invocations and still return the correct Id.
func Test_NewBuffer_threadSafe(t *testing.T) {
	// We're going to use a wait group to block each goroutine after it's prepared until they're all
	// ready to fire.
	numIter := 100
	var wg sync.WaitGroup
	wg.Add(numIter)

	dataCh := make(chan BufferId)

	// Prepare one goroutine to create a new buffer for each iteration.
	for i := 0; i < numIter; i++ {
		// Calculate the dimensions that could be used for this buffer.
		length := rand.Intn(20) + 1
		width := rand.Intn(20) + 1
		height := rand.Intn(20) + 1

		// Spin up a new goroutine. This will wait until all goroutines are ready to fire, then
		// create a new metal buffer and send its Id back to the main thread.
		go func() {
			wg.Wait()

			var bufferId BufferId
			var err error
			switch i % 3 {
			case 0:
				bufferId, _, err = NewBuffer1D[int32](length)
			case 1:
				bufferId, _, err = NewBuffer2D[int32](length, width)
			case 2:
				bufferId, _, err = NewBuffer3D[int32](length, width, height)
			}
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

	// Check that we received every Id in the sequence.
	idList := make([]BufferId, 0, len(idMap))
	for bufferId := range idMap {
		idList = append(idList, bufferId)
	}
	sort.Slice(idList, func(i, j int) bool { return idList[i] < idList[j] })
	require.Len(t, idList, numIter)
	for i := 0; i < numIter; i++ {
		require.Equal(t, nextMetalId-numIter+i, int(idList[i]))
	}
}
