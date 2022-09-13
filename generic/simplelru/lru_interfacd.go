//go:build go1.18
// +build go1.18

// Package simplelru provides simple LRU implementation based on build-in container/list.
package simplelru

// LRUCache is the interface for simple LRU cache.
type LRUCache[Key comparable, T any] interface {
	// Adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Add(key Key, value T) bool

	// Returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key Key) (value T, ok bool)

	// Checks if a key exists in cache without updating the recent-ness.
	Contains(key Key) (ok bool)

	// Returns key's value without updating the "recently used"-ness of the key.
	Peek(key Key) (value T, ok bool)

	// Removes a key from the cache.
	Remove(key Key) bool

	// Removes the oldest entry from cache.
	RemoveOldest() (Key, T, bool)

	// Returns the oldest entry from the cache. #key, value, isFound
	GetOldest() (Key, T, bool)

	// Returns a slice of the keys in the cache, from oldest to newest.
	Keys() []Key

	// Returns the number of items in the cache.
	Len() int

	// Clears all cache entries.
	Purge()

	// Resizes cache, returning number evicted
	Resize(int) int
}
