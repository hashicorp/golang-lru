package lru

import (
	"tslru"

	"github.com/hashicorp/golang-lru/simplelru"
)

// Fragmentation can reduce lock contention, but the hash function affects efficiency
type TSCache struct {
	lru simplelru.LRUCache
}

// New creates an LRU of the given size. total = size * bucketNum
func NewTsCache(size int) (c *TSCache, err error) {
	// create a cache with default settings
	lru, err := tslru.NewLRU(size)
	if err != nil {
		return nil, err
	}
	return &TSCache{lru: lru}, nil
}

// Purge is used to completely clear the cache.
func (c *TSCache) Purge() {
	c.lru.Purge()
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *TSCache) Add(key, value interface{}) (evicted bool) {
	evicted = c.lru.Add(key, value)
	return
}

// Get looks up a key's value from the cache.
func (c *TSCache) Get(key interface{}) (value interface{}, ok bool) {
	return c.lru.Get(key)
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *TSCache) Contains(key interface{}) bool {
	return c.lru.Contains(key)
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *TSCache) Peek(key interface{}) (value interface{}, ok bool) {
	return c.lru.Peek(key)
}

// ContainsOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *TSCache) ContainsOrAdd(key, value interface{}) (ok, evicted bool) {
	if c.lru.Contains(key) {
		return true, false
	}
	return false, c.lru.Add(key, value)
}

// PeekOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *TSCache) PeekOrAdd(key, value interface{}) (previous interface{}, ok, evicted bool) {
	previous, ok = c.lru.Peek(key)
	if ok {
		return previous, true, false
	}
	return nil, false, c.lru.Add(key, value)
}

// Remove removes the provided key from the cache.
func (c *TSCache) Remove(key interface{}) (present bool) {
	return c.lru.Remove(key)
}

// Resize changes the cache size.
func (c *TSCache) Resize(size int) (evicted int) {
	c.lru.Resize(size)
	return
}

//// RemoveOldest removes the oldest item from the cache.
//func (c *TSCache) RemoveOldest() (key, value interface{}, ok bool) {
//}

//// GetOldest returns the oldest entry
//func (c *TSCache) GetOldest() (key, value interface{}, ok bool) {
//	return c.lru.GetOldest()
//}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *TSCache) Keys() (ret []interface{}) {
	return c.lru.Keys()
}

// Len returns the number of items in the cache.
func (c *TSCache) Len() (ret int) {
	return c.lru.Len()
}
