package lru

import (
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
)

// Cache is a thread-safe fixed size LRU cache.
type Cache struct {
	lru  simplelru.LRUCache
	lock sync.RWMutex
}

// New creates an LRU of the given size.
func New(size int) (*Cache, error) {
	return NewWithEvict(size, nil)
}

// NewWithEvict constructs a fixed size cache with the given eviction
// callback.
func NewWithEvict(size int, onEvicted func(key interface{}, value interface{})) (*Cache, error) {
	lru, err := simplelru.NewLRU(size, simplelru.EvictCallback(onEvicted))
	if err != nil {
		return nil, err
	}
	c := &Cache{
		lru: lru,
	}
	return c, nil
}

// Purge is used to completely clear the cache.
func (c *Cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lru.Purge()
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *Cache) Add(key, value interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	evicted = c.lru.Add(key, value)
	return evicted
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	value, ok = c.lru.Get(key)
	return value, ok
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *Cache) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	containKey := c.lru.Contains(key)
	return containKey
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *Cache) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	value, ok = c.lru.Peek(key)
	return value, ok
}

// ContainsOrAdd checks if a key is in the cache  without updating the
// recent-ness or deleting it for being stale,  and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *Cache) ContainsOrAdd(key, value interface{}) (ok, evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.lru.Contains(key) {
		return true, false
	}
	evicted = c.lru.Add(key, value)
	return false, evicted
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key interface{}) (present bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	present = c.lru.Remove(key)
	return
}

// Resize changes the cache size.
func (c *Cache) Resize(size int) (evicted int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	evicted = c.lru.Resize(size)
	return evicted
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() (key interface{}, value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	key, value, ok = c.lru.RemoveOldest()
	return
}

// GetOldest returns the oldest entry
func (c *Cache) GetOldest() (key interface{}, value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	key, value, ok = c.lru.GetOldest()
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *Cache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	keys := c.lru.Keys()
	return keys
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	length := c.lru.Len()
	return length
}
