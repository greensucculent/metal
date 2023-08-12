//go:build darwin
// +build darwin

package metal

import (
	"fmt"
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
