// go:build darwin
//  +build darwin

#ifndef HEADER_METAL
#define HEADER_METAL

#include <stdlib.h>

// Functions that must be called once for every application
void metal_init();

// Functions that must be called once for every metal function
int metal_newFunction(const char *metalCode, const char *funcName,
                      const char **);
void metal_runFunction(int functionId, int width, int height, int depth,
                       int *bufferIds, int numBufferIds, const char **);

// Functions that must be called once for every buffer used as an argument to
// a metal function
int metal_newBuffer(int size, const char **);
void *metal_retrieveBuffer(int bufferId, const char **);

#endif