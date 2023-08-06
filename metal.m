// go:build darwin
//  +build darwin

// The process in this file largely follows the structure detailed in
// https://developer.apple.com/documentation/metal/performing_calculations_on_a_gpu.

#include "metal.h"
#include "cache.h"
#include "function.h"

id<MTLDevice> device;

// Log an error to console and optionally set target to the error message.
void logError(NSString *message, const char **target) {
  if (target != nil) {
    *target = [message UTF8String];
  }

  NSLog(@"%@", message);
}

// Initialize the default GPU. This should be called only once for the lifetime
// of the app.
void metal_init() {
  // Get the default MTLDevice (each GPU is assigned its own device).
  device = MTLCreateSystemDefaultDevice();
  NSCAssert(device != nil, @"Failed to find default GPU");
}

// Set up a new pipeline for executing the specified function in the provided
// MTL code on the default GPU. This returns an Id that must be used to actually
// run the function. This should be called only once for every function. If any
// error is encountered initializing the metal function, this returns 0 and sets
// an error message in error.
int metal_newFunction(const char *metalCode, const char *funcName,
                      const char **error) {
  // Set up a new function object to hold the various resources for the
  // pipeline.
  _function *function = function_newFunction();
  if (function == nil) {
    logError(@"Failed to initialize function", error);
    return 0;
  }

  // Create a new library of metal code, which will be used to get a
  // reference to the function we want to run on the GPU. Normnally, we
  // would use newDefaultLibrary here to automatically create a library from
  // all the .metal files in this package. However, because cgo doesn't have
  // that functionality, we need to use newLibraryWithSource:options:error
  // instead and supply the code to the new library directly.
  NSError *libraryError = nil;
  id<MTLLibrary> library =
      [device newLibraryWithSource:[NSString stringWithUTF8String:metalCode]
                           options:[MTLCompileOptions new]
                             error:&libraryError];
  if (library == nil) {
    logError(@"Failed to create library", error);
    NSLog(@"%@", libraryError);
    return 0;
  }

  // Get a reference to the function in the code that's now in the new library.
  // (Note that this is not executable yet. We need a pipeline in order to
  // actually run this function.)
  id<MTLFunction> metalFunc =
      [library newFunctionWithName:[NSString stringWithUTF8String:funcName]];
  if (metalFunc == nil) {
    logError(@"Failed to find function", error);
    return 0;
  }

  // Convert the function object we just created into a pipeline so we can run
  // the function. A pipeline contains the actual instructions/steps that the
  // GPU uses to execute the code.
  NSError *pipelineError = nil;
  function->pipeline =
      [device newComputePipelineStateWithFunction:metalFunc
                                            error:&pipelineError];
  if (function->pipeline == nil) {
    logError([NSString stringWithFormat:@"Failed to create pipeline: %@",
                                        pipelineError],
             error);
    return 0;
  }

  // Set up a command queue. This is what sends the work to the GPU.
  function->commandQueue = [device newCommandQueue];
  if (function->commandQueue == nil) {
    logError(@"Failed to set up command queue", error);
    return 0;
  }

  // Save the function for later use and return an Id referencing it.
  return cache_cache(function);
}

// Execute the computational process on the GPU. Each buffer is supplied as an
// argument to the metal code in the same order as the buffer Ids here. This is
// not thread-safe.
void metal_runFunction(int functionId, int width, int height, int depth,
                       int *bufferIds, int numBufferIds, const char **error) {
  // Fetch the function from the cache.
  _function *function = cache_retrieve(functionId);
  if (function == nil) {
    logError(@"Failed to retrieve function", error);
    return;
  }

  // Create a command buffer from the command queue in the pipeline. This will
  // hold the processing commands and move through the queue to the GPU.
  id<MTLCommandBuffer> commandBuffer = [function->commandQueue commandBuffer];
  if (commandBuffer == nil) {
    logError(@"Failed to set up command buffer", error);
    return;
  }

  // Set up an encoder to actually write the (compute pass) commands and
  // parameters to the command buffer we just created.
  id<MTLComputeCommandEncoder> encoder = [commandBuffer computeCommandEncoder];
  if (encoder == nil) {
    logError(@"Failed to set up compute encoder", error);
    return;
  }

  // Set the pipeline that the command will use.
  [encoder setComputePipelineState:function->pipeline];

  // Set the buffers that will be passed as the arguments to the function. The
  // indexes for the buffers here need to match their order in the function
  // declaration. We currently only support using the entire buffer without any
  // offsets, which could be used to, say, use one part of a buffer for one
  // function argument and the other part for a different argument.
  for (int i = 0; i < numBufferIds; i++) {
    // Retrieve the buffer for this Id.
    id<MTLBuffer> buffer = cache_retrieve(bufferIds[i]);
    if (buffer == nil) {
      logError(@"Failed to retrieve buffer", error);
      return;
    }

    // Add the buffer to the command with the appropriate index.
    [encoder setBuffer:buffer offset:0 atIndex:i];
  }

  // Specify how many threads we need to perform all the calculations (one
  // thread per calculation).
  MTLSize gridSize = MTLSizeMake(width, height, depth);

  // Figure out how many threads will be grouped together into each threadgroup.
  // There are two variables that are important here:
  //
  //     pipeline.threadExecutionWidth:
  //         Maximum number of threads that the GPU can execute at one time in
  //         parallel (aka thread warp size)
  //     pipeline.maxTotalThreadsPerThreadgroup:
  //         Maximum number of threads that can be bundled together into a
  //         threadgroup
  //
  // We're going to divide the threads conceptually into two dimensions and then
  // place them into a 3-dimensional grid with no height. The first dimension
  // will be the number of threads that can run at one time (the thread warp
  // size). The second dimension will be the maximum number of parallel thread
  // bundles.
  //
  // For more details on threads, grids, and threadgroup sizes, see
  // https://developer.apple.com/documentation/metal/compute_passes/calculating_threadgroup_and_grid_sizes.
  NSUInteger w = function->pipeline.threadExecutionWidth;
  NSUInteger h = function->pipeline.maxTotalThreadsPerThreadgroup / w;
  MTLSize threadgroupSize = MTLSizeMake(w, h, 1);

  // Set the grid into the encoder. (With this method, we don't need to
  // calculate the number of threadgroups for the grid.)
  [encoder dispatchThreads:gridSize threadsPerThreadgroup:threadgroupSize];

  // Mark that we're done encoding the buffer and can proceed with executing the
  // function.
  [encoder endEncoding];

  // Commit the command buffer to the command queue so that it gets picked up
  // and run on the GPU, and then wait for the calculations to finish.
  [commandBuffer commit];
  [commandBuffer waitUntilCompleted];
}

// Allocate a block of memory accessible to both the CPU and GPU that is large
// enough to hold the number of bytes specified. The buffer is cached and can be
// retrieved with the buffer Id that's returned. A buffer can be supplied as an
// argument to the metal function when the function is run. If any error is
// encountered creating the buffer, this returns 0 and sets an error
// message in error.
int metal_newBuffer(int size, const char **error) {
  id<MTLBuffer> buffer =
      [device newBufferWithLength:(size) options:MTLResourceStorageModeShared];
  if (buffer == nil) {
    logError(@"Failed to create buffer", error);
    return 0;
  }

  // Add the buffer to the buffer cache and return its unique Id.
  return cache_cache(buffer);
}

// Retrieve a buffer from the cache. If any error is encountered retrieving the
// buffer, this returns nil and sets an error message in error.
void *metal_retrieveBuffer(int bufferId, const char **error) {
  id<MTLBuffer> buffer = cache_retrieve(bufferId);
  if (buffer == nil) {
    logError(@"Failed to retrieve buffer", error);
    return nil;
  }

  return [buffer contents];
}
