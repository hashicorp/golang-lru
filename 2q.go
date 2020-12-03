package lru

import (
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
)

// TwoQueueCache is a thread-safe fixed size 2Q cache.
// 2Q is an enhancement over the standard LRU cache
// in that it tracks both frequently and recently used
// entries separately. This avoids a burst in access to new
// entries from evicting frequently used entries. It adds some
// additional tracking overhead to the standard LRU cache, and is
// computationally about 2x the cost, and adds some metadata over
// head. The ARCCache is similar, but does not require setting any
// parameters.
type TwoQueueCache struct {
	lru  *simplelru.TwoQueueLRU
	lock sync.RWMutex
}

// New2Q creates a new TwoQueueCache using the default
// values for the parameters.
func New2Q(size int) (*TwoQueueCache, error) {
	return New2QParams(size, simplelru.Default2QRecentRatio, simplelru.Default2QGhostEntries)
}

// New2QParams creates a new TwoQueueCache using the provided
// parameter values.
func New2QParams(size int, recentRatio, ghostRatio float64) (*TwoQueueCache, error) {
	lru, err := simplelru.New2QParams(size, recentRatio, ghostRatio)
	if err != nil {
		return nil, err
	}
	return &TwoQueueCache{
		lru: lru,
	}, nil
}

// Get looks up a key's value from the cache.
func (c *TwoQueueCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Get(key)
}

// Add adds a value to the cache, return evicted key/val if eviction happens.
func (c *TwoQueueCache) Add(key, value interface{}, evictedKeyVal ...*interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Add(key, value, evictedKeyVal...)
}

// Len returns the number of items in the cache.
func (c *TwoQueueCache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Len()
}

// Keys returns a slice of the keys in the cache.
// The frequently used keys are first in the returned slice.
func (c *TwoQueueCache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Keys()
}

// Remove removes the provided key from the cache.
func (c *TwoQueueCache) Remove(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Remove(key)
}

// Purge is used to completely clear the cache.
func (c *TwoQueueCache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lru.Purge()
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (c *TwoQueueCache) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (c *TwoQueueCache) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.lru.Peek(key)
}
