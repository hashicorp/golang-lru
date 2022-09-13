//go:build go1.18
// +build go1.18

package simplelru

import (
	"errors"
)

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback[Key comparable, T any] func(key Key, value T)

// LRU implements a non-thread safe fixed size LRU cache
type LRU[Key comparable, T any] struct {
	size      int
	evictList *List[*entry[Key, T]]
	items     map[Key]*Element[*entry[Key, T]]
	onEvict   EvictCallback[Key, T]
}

// entry is used to hold a value in the evictList
type entry[Key comparable, T any] struct {
	key   Key
	value T
}

// NewLRU constructs an LRU of the given size
func NewLRU[Key comparable, T any](size int, onEvict EvictCallback[Key, T]) (*LRU[Key, T], error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU[Key, T]{
		size:      size,
		evictList: New[*entry[Key, T]](),
		items:     make(map[Key]*Element[*entry[Key, T]]),
		onEvict:   onEvict,
	}
	return c, nil
}

// Purge is used to completely clear the cache.
func (c *LRU[Key, T]) Purge() {
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.Value.value)
		}
		delete(c.items, k)
	}
	c.evictList.Init()
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *LRU[Key, T]) Add(key Key, value T) (evicted bool) {
	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.value = value
		return false
	}

	// Add new item
	ent := &entry[Key, T]{key, value}
	entry := c.evictList.PushFront(ent)
	c.items[key] = entry

	evict := c.evictList.Len() > c.size
	// Verify size not exceeded
	if evict {
		c.removeOldest()
	}
	return evict
}

// Get looks up a key's value from the cache.
func (c *LRU[Key, T]) Get(key Key) (value T, ok bool) {
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		if ent.Value == nil {
			var tmp T
			return tmp, false
		}
		return ent.Value.value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU[Key, T]) Contains(key Key) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU[Key, T]) Peek(key Key) (value T, ok bool) {
	var ent *Element[*entry[Key, T]]
	if ent, ok = c.items[key]; ok {
		return ent.Value.value, true
	}

	return value, ok
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU[Key, T]) Remove(key Key) (present bool) {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU[Key, T]) RemoveOldest() (key Key, value T, ok bool) {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
		kv := ent.Value
		return kv.key, kv.value, true
	}

	return key, value, false
}

// GetOldest returns the oldest entry
func (c *LRU[Key, T]) GetOldest() (key Key, value T, ok bool) {
	ent := c.evictList.Back()
	if ent != nil {
		kv := ent.Value
		return kv.key, kv.value, true
	}
	return key, value, false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU[Key, T]) Keys() []Key {
	keys := make([]Key, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.key
		i++
	}
	return keys
}

// Len returns the number of items in the cache.
func (c *LRU[Key, T]) Len() int {
	return c.evictList.Len()
}

// Resize changes the cache size.
func (c *LRU[Key, T]) Resize(size int) (evicted int) {
	diff := c.Len() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.size = size
	return diff
}

// removeOldest removes the oldest item from the cache.
func (c *LRU[Key, T]) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *LRU[Key, T]) removeElement(e *Element[*entry[Key, T]]) {
	c.evictList.Remove(e)
	kv := e.Value
	delete(c.items, kv.key)
	if c.onEvict != nil {
		c.onEvict(kv.key, kv.value)
	}
}
