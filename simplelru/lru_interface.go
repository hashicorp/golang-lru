// Package simplelru provides simple LRU implementation based on build-in container/list.
package simplelru

// LRUCache is the interface for simple LRU cache.
type LRUCache[K comparable] interface {
	// Add adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	Add(key K, value any) bool

	// Get returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	Get(key K) (value any, ok bool)

	// Contains checks if a key exists in cache without updating the recent-ness.
	Contains(key K) (ok bool)

	// Peek returns key's value without updating the "recently used"-ness of the key.
	Peek(key K) (value any, ok bool)

	// Remove removes a key from the cache.
	Remove(key K) bool

	// RemoveOldest removes the oldest entry from cache.
	RemoveOldest() (K, any, bool)

	// GetOldest returns the oldest entry from the cache. #key, value, isFound
	GetOldest() (K, any, bool)

	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K

	// Len returns the number of items in the cache.
	Len() int

	// Purge clears all cache entries.
	Purge()

	// Resize resizes cache, returning number evicted
	Resize(int) int
}
