package metal_test

import (
	"log"

	"github.com/greensucculent/metal"
)

func ExampleNewFunction() {
	source := `
		#include <metal_stdlib>
		#include <metal_math>

		using namespace metal;

		kernel void sine(constant float *input, device float *result, uint pos [[thread_position_in_grid]]) {
			int index = pos;
			result[pos] = sin(input[pos]) * 0.01 * 0.01;
		}
	`

	functionId, err := metal.NewFunction(source, "sine")
	if err != nil {
		log.Fatalf("Unable to create metal function: %v", err)
	}

	// functionId is used to actually run the function later.
	_ = functionId
}

func ExampleNewBuffer1D() {
	// Create a 1-dimensional buffer with a width of 100. This will allocate 400 bytes (100 items *
	// sizeof(float32)).
	bufferId, buffer, err := metal.NewBuffer1D[float32](100)
	if err != nil {
		log.Fatalf("Unable to create metal buffer: %v", err)
	}

	// bufferId is used to reference the buffer when running a metal function later.
	_ = bufferId

	// buffer is used to load/retrieve data from the pipeline.
	_ = buffer
}

func ExampleNewBuffer2D() {
	// Create a 2-dimensional buffer with a width of 100 and a height of 20. This will allocate
	// 8,000 bytes (100 * 20 * sizeof(float32)).
	bufferId, buffer, err := metal.NewBuffer2D[float32](100, 20)
	if err != nil {
		log.Fatalf("Unable to create metal buffer: %v", err)
	}

	// bufferId is used to reference the buffer when running a metal function later.
	_ = bufferId

	// buffer is used to load/retrieve data from the pipeline.
	_ = buffer
}

func ExampleNewBuffer3D() {
	// Create a 3-dimensional buffer with a width of 100, a height of 20, and a depth of 2. This
	// will allocate 16,000 bytes (100 * 20 * 2 * sizeof(float32)).
	bufferId, buffer, err := metal.NewBuffer3D[float32](100, 20, 2)
	if err != nil {
		log.Fatalf("Unable to create metal buffer: %v", err)
	}

	// bufferId is used to reference the buffer when running a metal function later.
	_ = bufferId

	// buffer is used to load/retrieve data from the pipeline.
	_ = buffer
}
