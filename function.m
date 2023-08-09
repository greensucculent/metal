// go:build darwin
//  +build darwin

#include "function.h"
#include "cache.h"
#include "error.h"

// Get the name of the metal function with the provided function Id, or nil on
// error.
const char *function_name(int functionId) {
  // Fetch the function from the cache.
  _function *function = cache_retrieve(functionId);
  if (function == nil) {
    logError(nil, @"Failed to retrieve function");
    return nil;
  }

  return [[function->function name] UTF8String];
}