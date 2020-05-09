package simplelru

import (
	"errors"
	"sync"
	"time"
)

// noEvictionTTL - very long ttl to prevent eviction
const noEvictionTTL = time.Hour * 24 * 365 * 10

// EvictCallback is used to get a callback when a cache entry is evicted
type EvictCallback[K comparable, V any] func(key K, value V)

// LRU implements a thread-safe safe fixed size LRU cache
// or expirable cache if expirable creation function was used.
type LRU[K comparable, V any] struct {
	size      int
	evictList *lruList[K, V]
	items     map[K]*entry[K, V]
	onEvict   EvictCallback[K, V]

	// expirable options
	mu         sync.Mutex
	purgeEvery time.Duration
	ttl        time.Duration
	done       chan struct{}
}

// NewLRU constructs a thread safe LRU of the given size
func NewLRU[K comparable, V any](size int, onEvict EvictCallback[K, V]) (*LRU[K, V], error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}

	c := &LRU[K, V]{
		size:      size,
		evictList: newList[K, V](),
		items:     make(map[K]*entry[K, V]),
		onEvict:   onEvict,
		ttl:       noEvictionTTL,
	}
	return c, nil
}

// NewExpirableLRU returns a new thread-safe cache with expirable entries.
//
// Size parameter set to 0 makes cache of unlimited size, e.g. turns LRU off.
//
// Providing 0 TTL turns expiring off.
//
// Activates deleteExpired by purgeEvery duration.
// If MaxKeys and TTL are defined and PurgeEvery is zero, PurgeEvery will be set to 5 minutes.
func NewExpirableLRU[K comparable, V any](size int, onEvict EvictCallback[K, V], ttl, purgeEvery time.Duration) *LRU[K, V] {
	if size < 0 {
		size = 0
	}
	if ttl <= 0 {
		ttl = noEvictionTTL
	}

	res := LRU[K, V]{
		items:      make(map[K]*entry[K, V]),
		evictList:  newList[K, V](),
		ttl:        ttl,
		purgeEvery: purgeEvery,
		size:       size,
		onEvict:    onEvict,
		done:       make(chan struct{}),
	}

	// enable deleteExpired() running in separate goroutine for cache
	// with non-zero TTL and size defined
	if res.ttl != noEvictionTTL && (res.size > 0 || res.purgeEvery > 0) {
		if res.purgeEvery <= 0 {
			res.purgeEvery = time.Minute * 5 // non-zero purge enforced because size defined
		}
		go func(done <-chan struct{}) {
			ticker := time.NewTicker(res.purgeEvery)
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					res.mu.Lock()
					res.deleteExpired()
					res.mu.Unlock()
				}
			}
		}(res.done)
	}
	return &res
}

// Purge clears the cache completely.
func (c *LRU[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.items {
		if c.onEvict != nil {
			c.onEvict(k, v.value)
		}
		delete(c.items, k)
	}
	c.evictList.init()
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *LRU[K, V]) Add(key K, value V) (evicted bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.moveToFront(ent)
		ent.value = value
		ent.expiresAt = now.Add(c.ttl)
		return false
	}

	// Add new item
	c.items[key] = c.evictList.pushFront(key, value, now.Add(c.ttl))

	// Verify size not exceeded
	if c.size > 0 && len(c.items) > c.size {
		c.removeOldest()
		return true
	}
	return false
}

// Get looks up a key's value from the cache.
func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent, found := c.items[key]; found {
		// Expired item check
		if time.Now().After(ent.expiresAt) {
			return
		}
		c.evictList.moveToFront(ent)
		return ent.value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU[K, V]) Contains(key K) (ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok = c.items[key]
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU[K, V]) Peek(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent, found := c.items[key]; found {
		// Expired item check
		if time.Now().After(ent.expiresAt) {
			return
		}
		return ent.value, true
	}
	return
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU[K, V]) Remove(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent := c.evictList.back(); ent != nil {
		c.removeElement(ent)
		return ent.key, ent.value, true
	}
	return
}

// GetOldest returns the oldest entry
func (c *LRU[K, V]) GetOldest() (key K, value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent := c.evictList.back(); ent != nil {
		return ent.key, ent.value, true
	}
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keys()
}

// Len returns the number of items in the cache.
func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.evictList.length()
}

// Resize changes the cache size. Size of 0 doesn't resize the cache, as it means unlimited.
func (c *LRU[K, V]) Resize(size int) (evicted int) {
	if size <= 0 {
		c.size = 0
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	diff := c.evictList.length() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	c.size = size
	return diff
}

// Close cleans the cache and destroys running goroutines for expirable cache,
// and does nothing for non-expirable one.
func (c *LRU[K, V]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	close(c.done)
}

// removeOldest removes the oldest item from the cache. Has to be called with lock!
func (c *LRU[K, V]) removeOldest() {
	if ent := c.evictList.back(); ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache. Has to be called with lock!
func (c *LRU[K, V]) removeElement(e *entry[K, V]) {
	c.evictList.remove(e)
	delete(c.items, e.key)
	if c.onEvict != nil {
		c.onEvict(e.key, e.value)
	}
}

// deleteExpired deletes expired records. Has to be called with lock!
func (c *LRU[K, V]) deleteExpired() {
	for _, key := range c.keys() {
		if time.Now().After(c.items[key].expiresAt) {
			c.removeElement(c.items[key])
			continue
		}
	}
}

// keys returns a slice of the keys in the cache, from oldest to newest. Has to be called with lock!
func (c *LRU[K, V]) keys() []K {
	keys := make([]K, 0, len(c.items))
	for ent := c.evictList.back(); ent != nil; ent = ent.prevEntry() {
		keys = append(keys, ent.key)
	}
	return keys
}
