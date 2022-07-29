// Package simplelru provides simple LRU implementation based on build-in container/list.
package simplelru

// LRUCache is the interface for simple LRU cache.
type LRUCache interface {
	// Adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Add(key, value any) bool

	// Returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key any) (value any, ok bool)

	// Checks if a key exists in cache without updating the recent-ness.
	Contains(key any) (ok bool)

	// Returns key's value without updating the "recently used"-ness of the key.
	Peek(key any) (value any, ok bool)

	// Removes a key from the cache.
	Remove(key any) bool

	// Removes the oldest entry from cache.
	RemoveOldest() (any, any, bool)

	// Returns the oldest entry from the cache. #key, value, isFound
	GetOldest() (any, any, bool)

	// Returns a slice of the keys in the cache, from oldest to newest.
	Keys() []any

	// Returns the number of items in the cache.
	Len() int

	// Clears all cache entries.
	Purge()

	// Resizes cache, returning number evicted
	Resize(int) int
}
