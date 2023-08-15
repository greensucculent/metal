//go:build darwin
// +build darwin

package metal

import (
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed test/noop.metal
	sourceNoop string
	//go:embed test/transfer.metal
	sourceTransfer string
)

var (
	// nextMetalId tracks the Id that should be returned for the next metal resource. We're going to
	// use this to make sure the metal cache is working as expected. Every time a new metal function
	// or metal buffer is created, this should be incremented. Because this is a global variable,
	// all tests that create new metal resources must be run concurrently.
	nextMetalId = 1
)

// validId checks if the Id has the expected value.
func validId[T FunctionId | BufferId](id T) bool {
	ok := int(id) == nextMetalId
	if ok {
		nextMetalId++
	}

	return ok
}

// addId marks that another Id was returned for a metal resource.
func addId() {
	nextMetalId++
}

// Test_FunctionId_NewFunction tests that NewFunction either creates a new metal function or returns
// the expected message, depending on the conditions of each scenario.
func Test_FunctionId_NewFunction(t *testing.T) {
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
			source:   sourceTransfer,
			function: "",
			wantErr:  "Unable to set up metal function: Missing function name",
		},
		{
			// Valid source, invalid function name
			source:   sourceTransfer,
			function: "invalid",
			wantErr:  "Unable to set up metal function: Failed to find function 'invalid'",
		},
		{
			// Valid source, valid function name
			source:   sourceTransfer,
			function: "transfer",
		},
	}

	for i, scenario := range scenarios {
		t.Run(fmt.Sprintf("Scenario%02d", i+1), func(t *testing.T) {
			// Try to create a new metal function with the provided source and function name.
			functionId, err := NewFunction(scenario.source, scenario.function)

			// Check that the scenario's expected error and the actual error line up.
			if scenario.wantErr == "" {
				require.Nil(t, err, "Unable to create metal function: %s", err)
				require.True(t, functionId.Valid())
				require.True(t, validId(functionId))
			} else {
				require.NotNil(t, err)
				require.Equal(t, scenario.wantErr, err.Error())
				require.False(t, functionId.Valid())
			}
		})
	}

	// Create a range of new functions and test that the returned function Id is always incremented
	// by 1.
	for i := 0; i < 100; i++ {
		functionId, err := NewFunction(sourceTransfer, "transfer")
		require.Nil(t, err)
		require.True(t, validId(functionId))
	}
}

// Test_FunctionId_Valid tests that FunctionId's Valid method correctly identifies a valid function Id.
func Test_FunctionId_Valid(t *testing.T) {
	// A valid function Id has a positive value. Let's run through a bunch of numbers and test that
	// Valid always reports the correct status.
	for i := -100_00; i <= 100_000; i++ {
		function := FunctionId(i)

		if i > 0 {
			require.True(t, function.Valid())
		} else {
			require.False(t, function.Valid())
		}
	}
}

// Test_FunctionId_String tests that FunctionId's String method returns the correct function name.
func Test_FunctionId_String(t *testing.T) {
	// Test an uninitialized function.
	var function FunctionId
	require.Equal(t, "", function.String())

	// Test an invalid function.
	functionId, err := NewFunction("", "")
	require.NotNil(t, err)
	require.False(t, functionId.Valid())
	require.Equal(t, "", functionId.String())

	// Test a valid function.
	functionId, err = NewFunction(sourceTransfer, "transfer")
	require.Nil(t, err)
	require.True(t, functionId.Valid())
	require.True(t, validId(functionId))
	require.Equal(t, "transfer", functionId.String())
}

// Test_FunctionId_NewFunction_threadSafe tests that NewFunction can handle multiple parallel invocations and
// still return the correct function Id.
func Test_FunctionId_NewFunction_threadSafe(t *testing.T) {
	type data struct {
		functionId FunctionId
		wantName   string
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

			functionId, err := NewFunction(source, functionName)
			require.Nil(t, err, "Unable to create metal function %s: %s", functionName, err)

			dataCh <- data{
				functionId: functionId,
				wantName:   functionName,
			}
		}()

		// Mark that this goroutine is ready.
		wg.Done()
	}

	// Check that each function Id is unique and references the correct function.
	idMap := make(map[FunctionId]struct{})
	for i := 0; i < numIter; i++ {
		data := <-dataCh

		_, ok := idMap[data.functionId]
		require.False(t, ok)
		idMap[data.functionId] = struct{}{}

		haveName := data.functionId.String()
		require.Equal(t, data.wantName, haveName)

		addId()
	}

	// Check that we received every Id in the sequence.
	idList := make([]FunctionId, 0, len(idMap))
	for functionId := range idMap {
		idList = append(idList, functionId)
	}
	sort.Slice(idList, func(i, j int) bool { return idList[i] < idList[j] })
	require.Len(t, idList, numIter)
	for i := 0; i < numIter; i++ {
		require.Equal(t, nextMetalId-numIter+i, int(idList[i]))
	}
}

// Test_FunctionId_Run_invalid tests that FunctionId's Run method correctly handles invalid
// parameters.
func Test_FunctionId_Run_invalid(t *testing.T) {
	functionId, err := NewFunction(sourceNoop, "noop")
	require.Nil(t, err)
	require.True(t, validId(functionId))

	// Test calling Run with an invalid (uninitialized) Function.
	var emptyFunction FunctionId
	err = emptyFunction.Run(Grid{})
	require.NotNil(t, err)
	require.Equal(t, "Unable to run metal function: Failed to retrieve function", err.Error())

	// Test calling Run with a buffer Id for a buffer that doesn't exist.
	err = functionId.Run(Grid{}, BufferId(10000))
	require.NotNil(t, err)
	require.Equal(t, "Unable to run metal function: Failed to retrieve buffer 1/1 using Id 10000", err.Error())

	// Test calling Run with an invalid Grid.
	err = functionId.Run(Grid{X: -1, Y: -1, Z: -1})
	require.Nil(t, err)
}

// Test_FunctionId_Run_1D tests that FunctionId's Run method correctly runs a 1-dimensional
// computational process for small and large input sizes.
func Test_FunctionId_Run_1D(t *testing.T) {
	for _, numElems := range []int{100, 100_000, 100_000_000} {
		t.Run(strconv.Itoa(numElems), func(t *testing.T) {

			// Set up a metal function that simply transfers all inputs to the outputs.
			functionId, err := NewFunction(sourceTransfer, "transfer")
			require.Nil(t, err)
			require.True(t, validId(functionId))

			// Set up an input and output buffer.
			inputId, input, err := NewBuffer1D[float32](numElems)
			require.Nil(t, err)
			require.True(t, validId(inputId))
			outputId, output, err := NewBuffer1D[float32](numElems)
			require.Nil(t, err)
			require.True(t, validId(outputId))

			// Set some initial values for the input.
			for i := range input {
				input[i] = float32(i + 1)
			}

			// Run the function and test that all values were transferred from the input to the output.
			require.NotEqual(t, input, output)
			err = functionId.Run(Grid{X: numElems}, inputId, outputId)
			require.Nil(t, err)
			require.Equal(t, input, output)

			// Set some different values in the input and run the function again.
			for i := range input {
				input[i] = float32(i * i)
			}
			require.NotEqual(t, input, output)
			err = functionId.Run(Grid{X: numElems}, inputId, outputId)
			require.Nil(t, err)
			require.Equal(t, input, output)
		})
	}
}
