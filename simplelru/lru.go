// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package simplelru

import (
	"errors"

	"github.com/hashicorp/golang-lru/v2/internal"
)

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback[K comparable, V any] func(key K, value V)

// LRU implements a non-thread safe fixed size LRU cache
type LRU[K comparable, V any] struct {
	size      int
	evictList *internal.LruList[K, V]
	items     map[K]*internal.Entry[K, V]
	onEvict   EvictCallback[K, V]
	hand      *internal.Entry[K, V]
	useSieve  bool
}

// NewLRU constructs an LRU of the given size
func NewLRU[K comparable, V any](size int, onEvict EvictCallback[K, V]) (*LRU[K, V], error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}

	c := &LRU[K, V]{
		size:      size,
		evictList: internal.NewList[K, V](),
		items:     make(map[K]*internal.Entry[K, V]),
		onEvict:   onEvict,
	}

	return c, nil
}

// NewSieve constructs an LRU of the given size
func NewSieve[K comparable, V any](size int, onEvict EvictCallback[K, V]) (*LRU[K, V], error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}

	c := &LRU[K, V]{
		size:      size,
		evictList: internal.NewList[K, V](),
		items:     make(map[K]*internal.Entry[K, V]),
		onEvict:   onEvict,
		hand:      nil,
		useSieve:  true,
	}

	return c, nil
}

// Purge is used to completely clear the cache.
func (c *LRU[K, V]) Purge() {
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.Value)
		}
		delete(c.items, k)
	}
	c.evictList.Init()
}

// Add adds a value to the cache.  Returns true if an eviction occurred.
func (c *LRU[K, V]) Add(key K, value V) (evicted bool) {
	// Check for existing item
	if ent, ok := c.items[key]; ok {
		if c.useSieve {
			ent.Visited = true
		} else {
			c.evictList.MoveToFront(ent)
		}

		ent.Value = value
		return false
	}

	if c.useSieve {
		if c.evictList.Length() >= c.size {
			c.performSieveEviction()
			evicted = true
		}

		ent := c.evictList.PushFront(key, value)
		ent.Visited = false
		c.items[key] = ent
		return
	}

	// Add new item
	ent := c.evictList.PushFront(key, value)
	c.items[key] = ent

	evict := c.evictList.Length() > c.size
	// Verify size not exceeded
	if evict {
		c.removeOldest()
	}
	return evict
}

// Get looks up a key's value from the cache.
func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	if ent, ok := c.items[key]; ok {
		if c.useSieve {
			ent.Visited = true
		} else {
			c.evictList.MoveToFront(ent)
		}

		return ent.Value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU[K, V]) Contains(key K) (ok bool) {
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU[K, V]) Peek(key K) (value V, ok bool) {
	var ent *internal.Entry[K, V]
	if ent, ok = c.items[key]; ok {
		return ent.Value, true
	}
	return
}

// visited returns if the key is visited
func (c *LRU[K, V]) visited(key K) (present bool, visited bool) {
	var ent *internal.Entry[K, V]
	if ent, present = c.items[key]; present {
		visited = ent.Visited
	}

	return
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU[K, V]) Remove(key K) (present bool) {
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	if c.useSieve {
		return c.performSieveEviction()
	}

	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent)
		return ent.Key, ent.Value, true
	}
	return
}

// GetOldest returns the oldest entry
func (c *LRU[K, V]) GetOldest() (key K, value V, ok bool) {
	if c.useSieve {
		c.getSieveCandidate()
		if c.hand != nil {
			return c.hand.Key, c.hand.Value, true
		}

		return
	}

	if ent := c.evictList.Back(); ent != nil {
		return ent.Key, ent.Value, true
	}
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU[K, V]) Keys() []K {
	keys := make([]K, c.evictList.Length())
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		keys[i] = ent.Key
		i++
	}
	return keys
}

// Values returns a slice of the values in the cache, from oldest to newest.
func (c *LRU[K, V]) Values() []V {
	values := make([]V, len(c.items))
	i := 0
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		values[i] = ent.Value
		i++
	}
	return values
}

// Len returns the number of items in the cache.
func (c *LRU[K, V]) Len() int {
	return c.evictList.Length()
}

// Cap returns the capacity of the cache
func (c *LRU[K, V]) Cap() int {
	return c.size
}

// Resize changes the cache size.
func (c *LRU[K, V]) Resize(size int) (evicted int) {
	diff := c.Len() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		if c.useSieve {
			c.performSieveEviction()
		} else {
			c.removeOldest()
		}
	}

	c.size = size
	return diff
}

// performSieveEviction - runs a eviction by running Sieve Algorithm and returns the evicted value.
func (c *LRU[K, V]) performSieveEviction() (key K, value V, ok bool) {
	c.getSieveCandidate()
	if c.hand != nil {
		candidate := c.hand
		c.hand = c.hand.PrevEntry()
		c.removeElement(candidate)
		return candidate.Key, candidate.Value, true
	}

	return
}

// getSieveCandidate evicts an entry based on sieve algorithm.
func (c *LRU[K, V]) getSieveCandidate() {
	if c.Len() == 0 {
		return
	}

	if c.hand == nil {
		c.hand = c.evictList.Back()
	}

	for c.hand != nil && c.hand.Visited {
		c.hand.Visited = false
		c.hand = c.hand.PrevEntry()
		if c.hand == nil {
			c.hand = c.evictList.Back()
		}
	}
}

// removeOldest removes the oldest item from the cache.
func (c *LRU[K, V]) removeOldest() {
	if c.useSieve {
		c.performSieveEviction()
		return
	}

	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *LRU[K, V]) removeElement(e *internal.Entry[K, V]) {
	c.evictList.Remove(e)
	delete(c.items, e.Key)
	if c.onEvict != nil {
		c.onEvict(e.Key, e.Value)
	}
}
