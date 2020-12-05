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
	lru                    *simplelru.TwoQueueLRU
	evictedKey, evictedVal interface{}
	onEvictedCB            func(k, v interface{})
	lock                   sync.RWMutex
}

// New2Q creates a new TwoQueueCache using the default
// values for the parameters.
func New2Q(size int) (*TwoQueueCache, error) {
	return New2QParams(size, nil, simplelru.Default2QRecentRatio, simplelru.Default2QGhostEntries)
}

func New2QWithEvict(size int, onEvict func(k, v interface{})) (*TwoQueueCache, error) {
	return New2QParams(size, onEvict, simplelru.Default2QRecentRatio, simplelru.Default2QGhostEntries)
}

// New2QParams creates a new TwoQueueCache using the provided
// parameter values.
func New2QParams(size int, onEvict func(k, v interface{}), recentRatio, ghostRatio float64) (c *TwoQueueCache, err error) {
	c = &TwoQueueCache{onEvictedCB: onEvict}
	if onEvict != nil {
		onEvict = c.onEvicted
	}
	c.lru, err = simplelru.New2QParams(size, onEvict, recentRatio, ghostRatio)
	return
}

//evicted key/val will be saved and sent thru registered callback
//outside of critical section later
func (c *TwoQueueCache) onEvicted(k, v interface{}) {
	c.evictedKey = k
	c.evictedVal = v
}

// Get looks up a key's value from the cache.
func (c *TwoQueueCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Get(key)
}

// Add adds a value to the cache, return true if eviction happens.
func (c *TwoQueueCache) Add(key, value interface{}) (evicted bool) {
	var ke, ve interface{}
	c.lock.Lock()
	evicted = c.lru.Add(key, value)
	ke, ve = c.evictedKey, c.evictedVal
	c.evictedKey = nil
	c.evictedVal = nil
	c.lock.Unlock()
	if evicted && c.onEvictedCB != nil {
		c.onEvictedCB(ke, ve)
	}
	return
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
func (c *TwoQueueCache) Remove(key interface{}) (ok bool) {
	var ke, ve interface{}
	c.lock.Lock()
	ok = c.lru.Remove(key)
	ke, ve = c.evictedKey, c.evictedVal
	c.evictedKey = nil
	c.evictedVal = nil
	c.lock.Unlock()
	if ok && c.onEvictedCB != nil {
		c.onEvictedCB(ke, ve)
	}
	return
}

// Purge is used to completely clear the cache.
func (c *TwoQueueCache) Purge() {
	var keys, vals []interface{}
	c.lock.Lock()
	if c.onEvictedCB != nil {
		keys = c.lru.Keys()
		for _, k := range keys {
			val, _ := c.lru.Peek(k)
			vals = append(vals, val)
		}
	}
	c.lru.Purge()
	c.lock.Unlock()
	if c.onEvictedCB != nil {
		for i := 0; i < len(keys); i++ {
			c.onEvictedCB(keys[i], vals[i])
		}
	}
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
