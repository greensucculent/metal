#include <metal_stdlib>

using namespace metal;

kernel void transfer(device float *input, device float *result, uint index [[thread_position_in_grid]]) {
    result[index] = input[index];
}
