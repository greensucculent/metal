// go:build darwin
//  +build darwin

#import <Metal/Metal.h>

void **cache = nil;
int numItems = 0;

// The @synchronized directive requires an Objective-c object. We'll use this
// (even though we're not using the locking functionality) because it won't ever
// be touched for anything else.
NSLock *cacheLock = nil;

// Add an item to the cache.
int cache_cache(void *item) {
  NSCAssert(item != nil, @"Missing item to cache");

  int cacheId = 0;

  @synchronized(cacheLock) {
    numItems++;
    cache = realloc(cache, sizeof(void *) * numItems);

    cache[numItems - 1] = item;

    // A cache Id is an item's 1-based index in the cache.
    cacheId = numItems;
  }

  return cacheId;
}

// Retrieve an item from the cache.
void *cache_retrieve(int cacheId) {
  void *item = nil;

  @synchronized(cacheLock) {
    NSCAssert(cacheId >= 1, @"Invalid cache Id %d", cacheId);
    NSCAssert(cacheId <= numItems, @"Invalid cache Id %d", cacheId);

    // A cache Id is an item's 1-based index in the cache. We need to convert it
    // into a 0-based index to retrieve it from the cache.
    int index = cacheId - 1;

    item = cache[index];
  }

  return item;
}

// Remove an item from the cache.
void cache_remove(int cacheId) {
  @synchronized(cacheLock) {
    NSCAssert(cacheId >= 1, @"Invalid cache Id %d", cacheId);
    NSCAssert(cacheId <= numItems, @"Invalid cache Id %d", cacheId);

    // A cache Id is an item's 1-based index in the cache. We need to convert it
    // into a 0-based index to retrieve it from the cache.
    int index = cacheId - 1;

    // Set the item to nil.
    cache[index] = nil;
  }
}