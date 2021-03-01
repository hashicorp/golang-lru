package lru

import (
	"errors"

	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/hashicorp/golang-lru/tslru"
)

// TSCache is a thread-safe fixed size LRU cache.
type TSCache struct {
	// Fragmentation can reduce lock contention, but the hash function affects efficiency
	lru []simplelru.LRUCache
}

// NewTSCache creates an LRU of the given size.
// p is set to the number of cpu cores to improve concurrent write performance
func NewTSCache(size, p int) (c *TSCache, err error) {
	if p < 1 || size < 1 {
		return nil, errors.New("p cannot be less than 1")
	}
	if size < p {
		p = size
	}

	var ts = new(TSCache)
	ts.lru = make([]simplelru.LRUCache, p)

	for i := 1; i < p; i++ {
		// create a cache with default settings
		lru, err := tslru.NewLRU(size / p)
		if err != nil {
			return nil, err
		}
		ts.lru[i] = lru
	}

	lru, err := tslru.NewLRU(size/p + size%p)
	if err != nil {
		return nil, err
	}
	ts.lru[0] = lru

	return ts, nil
}

func (c *TSCache) bucket(key string) simplelru.LRUCache {
	if len(c.lru) == 1 {
		return c.lru[0]
	}
	return c.lru[djb(key)&uint32(len(c.lru)-1)]
}

// Purge is used to completely clear the cache.
func (c *TSCache) Purge() {
	for i := 0; i < len(c.lru); i++ {
		c.lru[i].Purge()
	}
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *TSCache) Add(key string, value interface{}) (evicted bool) {
	return c.bucket(key).Add(key, value)
}

// Get looks up a key's value from the cache.
func (c *TSCache) Get(key string) (value interface{}, ok bool) {
	return c.bucket(key).Get(key)
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *TSCache) Contains(key string) bool {
	return c.bucket(key).Contains(key)
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *TSCache) Peek(key string) (value interface{}, ok bool) {
	return c.bucket(key).Peek(key)
}

// ContainsOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *TSCache) ContainsOrAdd(key string, value interface{}) (ok, evicted bool) {
	if c.bucket(key).Contains(key) {
		return true, false
	}
	return false, c.bucket(key).Add(key, value)
}

// PeekOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *TSCache) PeekOrAdd(key string, value interface{}) (previous interface{}, ok, evicted bool) {
	previous, ok = c.bucket(key).Peek(key)
	if ok {
		return previous, true, false
	}
	return nil, false, c.bucket(key).Add(key, value)
}

// Remove removes the provided key from the cache.
func (c *TSCache) Remove(key string) (present bool) {
	return c.bucket(key).Remove(key)
}

// Resize changes the cache size.
func (c *TSCache) Resize(size int) (evicted int) {
	for i := 0; i < len(c.lru); i++ {
		evicted += c.lru[i].Resize(size)
	}
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *TSCache) Keys() (ret []string) {
	for i := 0; i < len(c.lru); i++ {
		for _, k := range c.lru[i].Keys() {
			ret = append(ret, k.(string))
		}
	}
	return
}

// Len returns the number of items in the cache.
func (c *TSCache) Len() (ret int) {
	for i := 0; i < len(c.lru); i++ {
		ret += c.lru[i].Len()
	}
	return
}

func djb(key string) uint32 {
	var h rune = 5381
	for _, r := range key {
		h = ((h << 5) + h) + r
	}
	return uint32(h)
}
