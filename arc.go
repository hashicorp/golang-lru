package lru

import (
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
)

// ARCCache is a thread-safe fixed size Adaptive Replacement Cache (ARC).
// ARC is an enhancement over the standard LRU cache in that tracks both
// frequency and recency of use. This avoids a burst in access to new
// entries from evicting the frequently used older entries. It adds some
// additional tracking overhead to a standard LRU cache, computationally
// it is roughly 2x the cost, and the extra memory overhead is linear
// with the size of the cache. ARC has been patented by IBM, but is
// similar to the TwoQueueCache (2Q) which requires setting parameters.
type ARCCache struct {
	lru  *simplelru.ARCLRU
	lock sync.RWMutex
}

// NewARC creates an ARC of the given size
func NewARC(size int) (*ARCCache, error) {
	lru, err := simplelru.NewARC(size)
	if err != nil {
		return nil, err
	}
	// Initialize the ARC
	c := &ARCCache{
		lru: lru,
	}

	return c, nil
}

// Get looks up a key's value from the cache.
func (c *ARCCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Get(key)
}

// Add adds a value to the cache, return evicted key/val if it happens.
func (c *ARCCache) Add(key, value interface{}, evictedKeyVal ...*interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Add(key, value, evictedKeyVal...)
}

// Len returns the number of cached entries
func (c *ARCCache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Len()
}

// Keys returns all the cached keys
func (c *ARCCache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Keys()
}

// Remove is used to purge a key from the cache
func (c *ARCCache) Remove(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Remove(key)
}

// Purge is used to clear the cache
func (c *ARCCache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lru.Purge()
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (c *ARCCache) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (c *ARCCache) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Peek(key)
}
