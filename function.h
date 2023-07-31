// go:build darwin
//  +build darwin

#ifndef HEADER_FUNCTION
#define HEADER_FUNCTION

#import <Metal/Metal.h>

// Structure of various metal resources needed to execute a computational
// process on the GPU. We have to bundle this in a header that cgo doesn't
// import because of a bug in LLVM that leads to a compilation error of "struct
// size calculation error off=8 bytesize=0".
typedef struct {
  id<MTLComputePipelineState> pipeline;
  id<MTLCommandQueue> commandQueue;
} _function;

_function *function_newFunction();

#endif