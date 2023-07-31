// go:build darwin
//  +build darwin

#include "function.h"

// Initialize a new metal function.
_function *function_newFunction() {
  _function *function = nil;

  function = malloc(sizeof(_function));
  NSCAssert(function != nil, @"Failed to initialize new function");

  return function;
}