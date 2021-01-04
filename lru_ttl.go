package lru

import (
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/simplelru"
)

// NOTE: this implementation will ensure that the cache will become eventually consistent.
// Expired items will stay in the cache until it is removed.
//
// When a GET is received on the expired item, the item is removed as part of the GET
// call. But, the other functions would still include the expired item in their result until
// it is removed by the cleanup routine.
//
// `Add` is the only call which will update the lastAccessTime of an item.

// CacheWithTTL implements thread safe fixed size LRU cache with TTL
type CacheWithTTL struct {
	*simplelru.LRU
	lock sync.RWMutex
	TTL  time.Duration
}

// cacheValue is a wrapper around the cache value to hold last accessed time
type cacheValue struct {
	value          interface{}
	lastAccessTime time.Time
}

// NewTTL constructs an LRU of the given size with the given TTL
func NewTTL(size int, ttl time.Duration) (simplelru.LRUCache, error) {
	return NewTTLWithEvict(size, ttl, nil)
}

// NewTTLWithEvict constructs an LRU of the given size with given TTL
// Also, sets up the evict function
func NewTTLWithEvict(size int, ttl time.Duration, onEvict simplelru.EvictCallback) (simplelru.LRUCache, error) {
	if size <= 0 {
		return nil, errors.New("Must provide a positive size")
	}

	lru, err := simplelru.NewLRU(size,
		func(k interface{}, v interface{}) {
			if onEvict != nil {
				onEvict(k, v.(cacheValue).value)
			}
		})
	if err != nil {
		return nil, err
	}

	lruWithTTL := &CacheWithTTL{LRU: lru, TTL: ttl}

	// clean expired items
	go lruWithTTL.cleanup()

	return lruWithTTL, nil
}

// Add adds the item to the cache. It also includes the `lastAccessTime` to the value.
// Life of an item can be increased by calling `Add` multiple times on the same key.
func (c *CacheWithTTL) Add(key, value interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.LRU.Add(key,
		cacheValue{
			value:          value,
			lastAccessTime: time.Now(),
		})
}

// Get looks up a key's value from the cache.
// Also, it unmarshals `lastAccessTime` from `Get` response
func (c *CacheWithTTL) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	val, ok := c.LRU.Get(key)
	// while the cleanup routine is catching will the other items, remove the item
	// if someone tries to access it through this GET call.
	if ok {
		if time.Now().After(val.(cacheValue).lastAccessTime.Add(c.TTL)) {
			c.LRU.Remove(key)
		} else {
			return val.(cacheValue).value, ok
		}
	}

	return nil, false
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
// Also, it unmarshals the `lastAccessTime` from the result
func (c *CacheWithTTL) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	val, ok := c.LRU.Peek(key)
	if ok {
		return val.(cacheValue).value, ok
	}
	return val, ok
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *CacheWithTTL) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.LRU.Contains(key)
}

// Purge is used to completely clear the cache.
func (c *CacheWithTTL) Purge() {
	c.lock.Lock()
	c.LRU.Purge()
	c.lock.Unlock()
}

// Remove removes the provided key from the cache.
func (c *CacheWithTTL) Remove(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.LRU.Remove(key)
}

// RemoveOldest removes the oldest item from the cache.
func (c *CacheWithTTL) RemoveOldest() (key interface{}, value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.LRU.RemoveOldest()
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *CacheWithTTL) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.LRU.Keys()
}

// Len returns the number of items in the cache.
func (c *CacheWithTTL) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.LRU.Len()
}

// cleanup deletes all the expired items
func (c *CacheWithTTL) cleanup() {
	ticker := time.NewTicker(2 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			for _, key := range c.Keys() {
				c.lock.Lock()
				val, ok := c.LRU.Get(key)
				c.lock.Unlock()

				if ok && time.Now().After(val.(cacheValue).lastAccessTime.Add(c.TTL)) {
					c.Remove(key)
				}
			}
		}
	}
}
