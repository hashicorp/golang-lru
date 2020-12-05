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
	lru                    *simplelru.ARCLRU
	evictedKey, evictedVal interface{}
	onEvictedCB            func(k, v interface{})
	lock                   sync.RWMutex
}

// NewARC creates an ARC of the given size
func NewARC(size int) (*ARCCache, error) {
	return NewARCWithEvict(size, nil)
}
func NewARCWithEvict(size int, onEvict func(k, v interface{})) (c *ARCCache, err error) {
	c = &ARCCache{onEvictedCB: onEvict}
	if onEvict != nil {
		onEvict = c.onEvicted
	}
	c.lru, err = simplelru.NewARCWithEvict(size, onEvict)
	return
}

//evicted key/val will be buffered and sent thru callback outside of critical section
func (c *ARCCache) onEvicted(k, v interface{}) {
	c.evictedKey = k
	c.evictedVal = v
}

//invoke callback outside of critical section to avoid dead-lock
func (c *ARCCache) sendEvicted() {
	if c.onEvictedCB != nil {
		c.onEvictedCB(c.evictedKey, c.evictedVal)
		c.evictedKey = nil
		c.evictedVal = nil
	}
}

// Get looks up a key's value from the cache.
func (c *ARCCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.lru.Get(key)
}

// Add adds a value to the cache, return evicted key/val if it happens.
func (c *ARCCache) Add(key, value interface{}) (evicted bool) {
	c.lock.Lock()
	evicted = c.lru.Add(key, value)
	c.lock.Unlock()
	c.sendEvicted()
	return
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
func (c *ARCCache) Remove(key interface{}) (ok bool) {
	c.lock.Lock()
	ok = c.lru.Remove(key)
	c.lock.Unlock()
	c.sendEvicted()
	return
}

// Purge is used to clear the cache
func (c *ARCCache) Purge() {
	var keys, vals []interface{}
	c.lock.Lock()
	if c.onEvicted != nil {
		keys = c.lru.Keys()
		for _, k := range keys {
			val, _ := c.lru.Peek(k)
			vals = append(vals, val)
		}
	}
	c.lru.Purge()
	c.lock.Unlock()
	if c.onEvicted != nil {
		for i := 0; i < len(keys); i++ {
			c.onEvicted(keys[i], vals[i])
		}
	}

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
