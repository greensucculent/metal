//go:build darwin
// +build darwin

package metal

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Metal source code common to all functions
var sourceCommon = `
#include <metal_stdlib>

using namespace metal;

`

// Metal source code for the function "transfer"
var sourceTransfer string = `
kernel void transfer(device float *input, device float *result, uint index [[thread_position_in_grid]]) {
    result[index] = input[index];
}
`

var (
	idCnt = 0
)

type subtest struct {
	name string
	f    func(*testing.T)
}

// runSubtests sequentially runs a list of subtests.
func runSubtests(t *testing.T, subtests []subtest) {
	for _, subtest := range subtests {
		t.Run(subtest.name, func(t *testing.T) {
			subtest.f(t)
		})
	}
}

// Test_Function is the handler for the Function subtests.
func Test_Function(t *testing.T) {
	runSubtests(t, []subtest{
		{"NewFunction", subtest_Function_NewFunction},
		{"Valid", subtest_Function_Valid},
		{"Id", subtest_Function_Id},
		{"Name", subtest_Function_Name},
		{"ThreadSafe", subtest_Function_ThreadSafe},
	})
}

// subtest_Function_NewFunction is a subtest for Function. It tests that NewFunction either creates a
// new metal function or returns the expected message, depending on the conditions of each scenario.
func subtest_Function_NewFunction(t *testing.T) {
	type scenario struct {
		source   string
		function string
		wantErr  string
	}

	scenarios := []scenario{
		{
			// No source or function name
			wantErr: "Unable to set up metal function: Missing metal code",
		},
		{
			// Invalid source, no function name
			source:  "invalid",
			wantErr: "Unable to set up metal function: Missing function name",
		},
		{
			// No source, invalid function name
			function: "invalid",
			wantErr:  "Unable to set up metal function: Missing metal code",
		},
		{
			// Invalid source, invalid function name
			source:   "invalid",
			function: "invalid",
			wantErr:  "Unable to set up metal function: Failed to create library (see console log)",
		},
		{
			// Valid source, no function name
			source:   sourceCommon + sourceTransfer,
			function: "",
			wantErr:  "Unable to set up metal function: Missing function name",
		},
		{
			// Valid source, invalid function name
			source:   sourceCommon + sourceTransfer,
			function: "invalid",
			wantErr:  "Unable to set up metal function: Failed to find function 'invalid'",
		},
		{
			// Valid source, valid function name
			source:   sourceCommon + sourceTransfer,
			function: "transfer",
		},
	}

	for i, scenario := range scenarios {
		t.Run(fmt.Sprintf("Scenario%02d", i+1), func(t *testing.T) {
			// Try to create a new metal function with the provided source and function name.
			function, err := NewFunction(scenario.source, scenario.function)

			// Check that the scenario's expected error and the actual error line up.
			if scenario.wantErr == "" {
				require.Nil(t, err, "Unable to create metal function: %s", err)
				require.True(t, function.Valid())
				idCnt++
			} else {
				require.NotNil(t, err)
				require.Equal(t, scenario.wantErr, err.Error())
				require.False(t, function.Valid())
			}
		})
	}
}

// subtest_Function_Valid is a subtest for Function. It tests that Function's Valid method correctly
// identifies a valid function.
func subtest_Function_Valid(t *testing.T) {
	// A valid Function has a positive Id. Let's run through a bunch of numbers and test that Valid
	// always reports the correct status.
	for i := -100_00; i <= 100_000; i++ {
		var function Function
		function.id = i

		if i > 0 {
			require.True(t, function.Valid())
		} else {
			require.False(t, function.Valid())
		}
	}
}

// subtest_Function_Id is a subtest for Function. It tests that Function's id field has the correct
// value for a variety of scenarios.
func subtest_Function_Id(t *testing.T) {
	// Invalid configuration: Id should be 0.
	function, err := NewFunction("", "")
	require.NotNil(t, err)
	require.Equal(t, 0, function.id)

	// Valid configuration: Id should be idCnt + 1, indicating a metal function was created and
	// added to the cache successfully.
	function, err = NewFunction(sourceCommon+sourceTransfer, "transfer")
	require.Nil(t, err)
	require.Equal(t, idCnt+1, function.id)
	idCnt++

	// Valid configuration: Id should be idCnt + 1, indicating a metal function was created and
	// added to the cache successfully.
	function, err = NewFunction(sourceCommon+sourceTransfer, "transfer")
	require.Nil(t, err)
	require.Equal(t, idCnt+1, function.id)
	idCnt++

	// Create a range of new functions and test that the returned function Id is always incremented
	// by 1.
	for i := 0; i < 100; i++ {
		function, err := NewFunction(sourceCommon+sourceTransfer, "transfer")
		require.Nil(t, err)
		require.Equal(t, idCnt+1, function.id)
		idCnt++
	}
}

// subtest_Function_Name is a subtest for Function. It tests that Function's String method returns the
// correct function name.
func subtest_Function_Name(t *testing.T) {
	// Test an uninitialized function.
	var function Function
	require.Equal(t, "", function.String())

	// Test an invalid function.
	function, err := NewFunction("", "")
	require.NotNil(t, err)
	require.False(t, function.Valid())
	require.Equal(t, "", function.String())

	// Test a valid function.
	function, err = NewFunction(sourceCommon+sourceTransfer, "transfer")
	require.Nil(t, err)
	require.True(t, function.Valid())
	require.Equal(t, "transfer", function.String())
	idCnt++
}

// subtest_Function_ThreadSafe is a subtest for Function. It tests that NewFunction can handle
// multiple parallel invocations and still return the correct Id.
func subtest_Function_ThreadSafe(t *testing.T) {
	type data struct {
		function Function
		wantName string
	}

	// We're going to use a wait group to block each goroutine after it's prepared until they're all
	// ready to fire.
	numIter := 100
	var wg sync.WaitGroup
	wg.Add(numIter)

	dataCh := make(chan data)

	// Prepare one goroutine to create a new function for each iteration.
	for i := 0; i < numIter; i++ {
		// Build the mock function name and mock metal code.
		functionName := fmt.Sprintf("abc_%d", i+1)
		source := fmt.Sprintf("kernel void %s() {}", functionName)

		// Spin up a new goroutine. This will wait until all goroutines are ready to fire, then
		// create a new metal function and send it back to the main thread.
		go func() {
			wg.Wait()

			function, err := NewFunction(source, functionName)
			require.Nil(t, err, "Unable to create metal function %s: %s", functionName, err)

			dataCh <- data{
				function: function,
				wantName: functionName,
			}
		}()

		// Mark that this goroutine is ready.
		wg.Done()
	}

	// Check that each function Id is unique and references the correct function.
	idMap := make(map[int]struct{})
	for i := 0; i < numIter; i++ {
		data := <-dataCh

		_, ok := idMap[data.function.id]
		require.False(t, ok)
		idMap[data.function.id] = struct{}{}

		haveName := data.function.String()
		require.Equal(t, data.wantName, haveName)

		idCnt++
	}
}

// Test_BufferId is the handler for the BufferId subtests.
func Test_BufferId(t *testing.T) {
	runSubtests(t, []subtest{
		{"NewBuffer", subtest_BufferId_NewBuffer},
		{"Valid", subtest_BufferId_Valid},
		{"ThreadSafe", subtest_BufferId_ThreadSafe},
	})
}

// subtest_BufferId_NewBuffer is a subtest for BufferId. It tests that NewBuffer creates a new metal
// buffer with the expected underlying type and number of elements.
func subtest_BufferId_NewBuffer(t *testing.T) {

	// Invalid configuration (no elements).
	bufferId, buffer, err := NewBuffer[int](0)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer)

	// Invalid configuration (negative elements).
	bufferId, buffer, err = NewBuffer[int](-1)
	require.NotNil(t, err)
	require.Equal(t, "Invalid number of elements", err.Error())
	require.Equal(t, BufferId(0), bufferId)
	require.Nil(t, buffer)

	testNewBuffer(t, func(i int) bool { return i%2 == 1 })
	testNewBuffer(t, func(i int) byte { return byte(i) })
	testNewBuffer(t, func(i int) rune { return rune(i) })
	testNewBuffer(t, func(i int) string { return strconv.Itoa(i) })
	testNewBuffer(t, func(i int) uint8 { return uint8(i) })
	testNewBuffer(t, func(i int) uint16 { return uint16(i) })
	testNewBuffer(t, func(i int) uint32 { return uint32(i) })
	testNewBuffer(t, func(i int) uint64 { return uint64(i) })
	testNewBuffer(t, func(i int) int8 { return int8(i) })
	testNewBuffer(t, func(i int) int16 { return int16(i) })
	testNewBuffer(t, func(i int) int32 { return int32(i) })
	testNewBuffer(t, func(i int) int64 { return int64(i) })
	testNewBuffer(t, func(i int) uint { return uint(i) })
	testNewBuffer(t, func(i int) int { return i })
	testNewBuffer(t, func(i int) uintptr { return uintptr(i) })
	testNewBuffer(t, func(i int) *int { return &i })
	testNewBuffer(t, func(i int) float32 { return float32(i) })
	testNewBuffer(t, func(i int) float64 { return float64(i) })
	testNewBuffer(t, func(i int) complex64 { return complex(float32(i), 0) })
	testNewBuffer(t, func(i int) complex128 { return complex(float64(i), 0) })
	testNewBuffer(t, func(i int) [3]int { return [3]int{i + 1, i + 2, i + 3} })
	testNewBuffer(t, func(i int) []int { return []int{i + 1, i + 2, i + 3} })
	testNewBuffer(t, func(i int) map[int]int { return map[int]int{i: i + 1} })

	type MyStruct struct {
		i int
		s string
	}
	testNewBuffer(t, func(i int) MyStruct { return MyStruct{i, strconv.Itoa(i)} })

	type MyInterface interface {
		Method1(int) string
		Method2(int) string
	}
	testNewBuffer(t, func(i int) MyInterface { var iface MyInterface; return iface })
}

// testNewBuffer is a helper to test buffer creation for a variety of types.
func testNewBuffer[T any](t *testing.T, converter func(int) T) {
	var a T
	t.Run(fmt.Sprintf("%T", a), func(t *testing.T) {

		bufferId, buffer, err := NewBuffer[T](10)
		require.Nil(t, err, "Unable to create metal buffer: %s", err)
		require.Equal(t, BufferId(idCnt+1), bufferId)
		require.Len(t, buffer, 10)
		require.Equal(t, len(buffer), cap(buffer))
		idCnt++

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

// subtest_BufferId_Valid is a subtest for BufferId. It tests that BufferId's Valid method correctly
// identifies a valid BufferId.
func subtest_BufferId_Valid(t *testing.T) {
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

// subtest_BufferId_ThreadSafe is a subtest for BufferId. It tests that NewBuffer can handle
// multiple parallel invocations and still return the correct Id.
func subtest_BufferId_ThreadSafe(t *testing.T) {
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

			bufferId, _, err := NewBuffer[int](length)
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
	}
}
