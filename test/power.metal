#include <metal_stdlib>

using namespace metal;

kernel void power(constant float *input, device float *result, uint pos [[thread_position_in_grid]]) {
    int index = pos;
    result[pos] = input[pos] * input[pos] * input[pos];
}