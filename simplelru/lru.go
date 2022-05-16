package simplelru

import (
	"errors"

	list "github.com/bahlo/generic-list-go"
)

// LRU implements a non-thread safe fixed size LRU cache
type LRU[K comparable, T any] struct {
	size      int
	evictList *list.List[*entry[K, T]]
	items     map[K]*list.Element[*entry[K, T]]
	onEvict   func(key K, value T)
}

// entry is used to hold a value in the evictList
type entry[K comparable, T any] struct {
	key   K
	value T
}

// NewLRU constructs an LRU of the given size
func NewLRU[K comparable, T any](size int, onEvict func(key K, value T)) (*LRU[K, T], error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU[K, T]{
		size:      size,
		evictList: list.New[*entry[K, T]](),
		items:     make(map[K]*list.Element[*entry[K, T]]),
		onEvict:   onEvict,
	}
	return c, nil
}

// Purge is used to completely clear the cache.
func (c *LRU[K, T]) Purge() {
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.Value.value)
		}
		delete(c.items, k)
	}
	c.evictList.Init()
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *LRU[K, T]) Add(key K, value T) (evicted bool) {
	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		ent.Value.value = value
		return false
	}

	// Add new item
	ent := &entry[K, T]{key, value}
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
func (c *LRU[K, T]) Get(key K) (value T, ok bool) {
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		if ent.Value == nil {
			var empty T
			return empty, false
		}
		return ent.Value.value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU[K, T]) Contains(key K) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU[K, T]) Peek(key K) (value T, ok bool) {
	var ent *list.Element[*entry[K, T]]
	if ent, ok = c.items[key]; ok {
		return ent.Value.value, true
	}
	var empty T
	return empty, ok
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU[K, T]) Remove(key K) (present bool) {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU[K, T]) RemoveOldest() (key K, value T, ok bool) {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
		kv := ent.Value
		return kv.key, kv.value, true
	}
	var emptyK K
	var emptyV T
	return emptyK, emptyV, false
}

// GetOldest returns the oldest entry
func (c *LRU[K, T]) GetOldest() (key K, value T, ok bool) {
	ent := c.evictList.Back()
	if ent != nil {
		kv := ent.Value
		return kv.key, kv.value, true
	}
	var emptyK K
	var emptyV T
	return emptyK, emptyV, false
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU[K, T]) Keys() []K {
	keys := make([]K, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys[i] = ent.Value.key
		i++
	}
	return keys
}

// Len returns the number of items in the cache.
func (c *LRU[K, T]) Len() int {
	return c.evictList.Len()
}

// Resize changes the cache size.
func (c *LRU[K, T]) Resize(size int) (evicted int) {
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
func (c *LRU[K, T]) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *LRU[K, T]) removeElement(e *list.Element[*entry[K, T]]) {
	c.evictList.Remove(e)
	kv := e.Value
	delete(c.items, kv.key)
	if c.onEvict != nil {
		c.onEvict(kv.key, kv.value)
	}
}
