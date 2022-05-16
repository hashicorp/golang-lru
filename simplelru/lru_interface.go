// Package simplelru provides simple LRU implementation based on build-in container/list.
package simplelru

// LRUCache is the interface for simple LRU cache.
type LRUCache[K comparable, T any] interface {
	// Adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Add(key K, value T) bool

	// Returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key K) (value T, ok bool)

	// Checks if a key exists in cache without updating the recent-ness.
	Contains(key K) (ok bool)

	// Returns key's value without updating the "recently used"-ness of the key.
	Peek(key K) (value T, ok bool)

	// Removes a key from the cache.
	Remove(key K) bool

	// Removes the oldest entry from cache.
	RemoveOldest() (K, T, bool)

	// Returns the oldest entry from the cache. #key, value, isFound
	GetOldest() (K, T, bool)

	// Returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K

	// Returns the number of items in the cache.
	Len() int

	// Clears all cache entries.
	Purge()

	// Resizes cache, returning number evicted
	Resize(int) int
}
