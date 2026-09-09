// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package expirable

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/v2/internal"
)

// EvictCallback is used to get a callback when a cache entry is evicted.
//
// Callbacks run in the goroutine that caused the eviction: for explicit
// removals (Add, Remove, RemoveOldest, Purge, Resize) they complete before
// the method returns; for expired entries they run in the background cleanup
// goroutine. They are always invoked outside the cache's internal lock, so
// they may safely re-enter the cache (e.g. call Get or Len). Because no lock
// is held while a callback runs, other goroutines may have mutated the cache
// by then, so callbacks must not assume they observe a consistent snapshot
// of the cache.
type EvictCallback[K comparable, V any] func(key K, value V)

// LRU implements a thread-safe LRU with expirable entries.
type LRU[K comparable, V any] struct {
	size      int
	evictList *internal.LruList[K, V]
	items     map[K]*internal.Entry[K, V]
	onEvict   EvictCallback[K, V]

	// expirable options
	mu   sync.Mutex
	ttl  time.Duration
	done chan struct{}

	// buckets for expiration
	buckets []bucket[K, V]
	// uint8 because it's number between 0 and numBuckets
	nextCleanupBucket uint8

	cleanupStopped bool
}

// bucket is a container for holding entries to be expired
type bucket[K comparable, V any] struct {
	entries     map[K]*internal.Entry[K, V]
	newestEntry time.Time
}

// noEvictionTTL - very long ttl to prevent eviction
const noEvictionTTL = time.Hour * 24 * 365 * 10

// because of uint8 usage for nextCleanupBucket, should not exceed 256.
// casting it as uint8 explicitly requires type conversions in multiple places
const numBuckets = 100

// NewLRU returns a new thread-safe cache with expirable entries.
//
// Size parameter set to 0 makes cache of unlimited size, e.g. turns LRU mechanism off.
//
// Providing 0 TTL turns expiring off.
//
// onEvict, if non-nil, is invoked for each evicted entry outside the cache's
// internal lock; see EvictCallback for the guarantees callbacks get.
//
// Delete expired entries every 1/100th of ttl value. Goroutine which deletes expired entries runs indefinitely.
func NewLRU[K comparable, V any](size int, onEvict EvictCallback[K, V], ttl time.Duration) *LRU[K, V] {
	if size < 0 {
		size = 0
	}
	if ttl <= 0 {
		ttl = noEvictionTTL
	}

	res := LRU[K, V]{
		ttl:            ttl,
		size:           size,
		evictList:      internal.NewList[K, V](),
		items:          make(map[K]*internal.Entry[K, V]),
		onEvict:        onEvict,
		done:           make(chan struct{}),
		cleanupStopped: false,
	}

	// initialize the buckets
	res.buckets = make([]bucket[K, V], numBuckets)
	for i := 0; i < numBuckets; i++ {
		res.buckets[i] = bucket[K, V]{entries: make(map[K]*internal.Entry[K, V])}
	}

	res.startGoroutine()
	return &res
}

// evictedEntry is a snapshot of a removed entry, kept so that the onEvict
// callback can be invoked after the lock is released.
type evictedEntry[K comparable, V any] struct {
	key   K
	value V
}

// fireCallbacks invokes the onEvict callback for each evicted entry.
// It must be called without holding c.mu so that callbacks can safely
// re-enter the cache.
func (c *LRU[K, V]) fireCallbacks(evicted []evictedEntry[K, V]) {
	if c.onEvict == nil {
		return
	}
	for _, e := range evicted {
		c.onEvict(e.key, e.value)
	}
}

// Purge clears the cache completely.
// onEvict is called for each evicted key, outside the cache's internal lock.
func (c *LRU[K, V]) Purge() {
	var evicted []evictedEntry[K, V]
	c.mu.Lock()
	for k, v := range c.items {
		delete(c.items, k)
		if c.onEvict != nil {
			evicted = append(evicted, evictedEntry[K, V]{key: k, value: v.Value})
		}
	}
	for _, b := range c.buckets {
		for _, ent := range b.entries {
			delete(b.entries, ent.Key)
		}
	}
	c.evictList.Init()
	c.mu.Unlock()
	c.fireCallbacks(evicted)
}

// Add adds a value to the cache. Returns true if an eviction occurred.
// Returns false if there was no eviction: the item was already in the cache,
// or the size was not exceeded.
// If an eviction occurred, onEvict is invoked for the evicted entry outside
// the cache's internal lock.
func (c *LRU[K, V]) Add(key K, value V) (evicted bool) {
	var evictedEntries []evictedEntry[K, V]
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		c.fireCallbacks(evictedEntries)
	}()
	now := time.Now()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		c.removeFromBucket(ent) // remove the entry from its current bucket as expiresAt is renewed
		ent.Value = value
		ent.ExpiresAt = now.Add(c.ttl)
		c.addToBucket(ent)
		return false
	}

	// Add new item
	ent := c.evictList.PushFrontExpirable(key, value, now.Add(c.ttl))
	c.items[key] = ent
	c.addToBucket(ent) // adds the entry to the appropriate bucket and sets entry.expireBucket

	evict := c.size > 0 && c.evictList.Length() > c.size
	// Verify size not exceeded
	if evict {
		c.removeOldest(&evictedEntries)
	}
	return evict
}

// AddIf adds a value to the cache if the key is not already present, or if
// replace returns true given the existing and new values. The replace function
// is not called when the key is absent. If replace is nil, an existing value
// is left unchanged.
//
// replace is invoked while the cache lock is held and must not call methods
// on the cache.
//
// Returns whether the cache was updated and whether an eviction occurred.
// Replacing an existing entry renews its TTL. A rejected update does not
// change recency or TTL. If an eviction occurred, onEvict is invoked for the
// evicted entry outside the cache's internal lock.
func (c *LRU[K, V]) AddIf(key K, value V, replace func(old V, new V) bool) (updated, evicted bool) {
	var evictedEntries []evictedEntry[K, V]
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		c.fireCallbacks(evictedEntries)
	}()
	now := time.Now()

	if ent, ok := c.items[key]; ok {
		if replace == nil || !replace(ent.Value, value) {
			return false, false
		}
		c.evictList.MoveToFront(ent)
		c.removeFromBucket(ent) // remove the entry from its current bucket as expiresAt is renewed
		ent.Value = value
		ent.ExpiresAt = now.Add(c.ttl)
		c.addToBucket(ent)
		return true, false
	}

	ent := c.evictList.PushFrontExpirable(key, value, now.Add(c.ttl))
	c.items[key] = ent
	c.addToBucket(ent)

	evict := c.size > 0 && c.evictList.Length() > c.size
	if evict {
		c.removeOldest(&evictedEntries)
	}
	return true, evict
}

// Get looks up a key's value from the cache.
func (c *LRU[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ent *internal.Entry[K, V]
	if ent, ok = c.items[key]; ok {
		// Expired item check
		if !c.cleanupStopped && time.Now().After(ent.ExpiresAt) {
			return value, false
		}
		c.evictList.MoveToFront(ent)
		return ent.Value, true
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
	var ent *internal.Entry[K, V]
	if ent, ok = c.items[key]; ok {
		// Expired item check
		if !c.cleanupStopped && time.Now().After(ent.ExpiresAt) {
			return value, false
		}
		return ent.Value, true
	}
	return
}

// Remove removes the provided key from the cache, returning if the
// key was contained. If it was, onEvict is invoked for the removed
// entry outside the cache's internal lock.
func (c *LRU[K, V]) Remove(key K) bool {
	var evictedEntries []evictedEntry[K, V]
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		c.fireCallbacks(evictedEntries)
	}()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent, &evictedEntries)
		return true
	}
	return false
}

// RemoveOldest removes the oldest item from the cache.
// If there was one, onEvict is invoked for it outside the cache's internal lock.
func (c *LRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	var evictedEntries []evictedEntry[K, V]
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		c.fireCallbacks(evictedEntries)
	}()
	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent, &evictedEntries)
		return ent.Key, ent.Value, true
	}
	return
}

// GetOldest returns the oldest entry
func (c *LRU[K, V]) GetOldest() (key K, value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent := c.evictList.Back(); ent != nil {
		return ent.Key, ent.Value, true
	}
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
// Expired entries are filtered out.
func (c *LRU[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]K, 0, len(c.items))
	now := time.Now()
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		if !c.cleanupStopped && now.After(ent.ExpiresAt) {
			continue
		}
		keys = append(keys, ent.Key)
	}
	return keys
}

// Values returns a slice of the values in the cache, from oldest to newest.
// Expired entries are filtered out.
func (c *LRU[K, V]) Values() []V {
	c.mu.Lock()
	defer c.mu.Unlock()
	values := make([]V, 0, len(c.items))
	now := time.Now()
	for ent := c.evictList.Back(); ent != nil; ent = ent.PrevEntry() {
		if !c.cleanupStopped && now.After(ent.ExpiresAt) {
			continue
		}
		values = append(values, ent.Value)
	}
	return values
}

// Len returns the number of items in the cache.
func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.evictList.Length()
}

// Resize changes the cache size. Size of 0 means unlimited.
// onEvict is invoked for each evicted entry outside the cache's internal lock.
func (c *LRU[K, V]) Resize(size int) (evicted int) {
	var evictedEntries []evictedEntry[K, V]
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		c.fireCallbacks(evictedEntries)
	}()
	if size <= 0 {
		c.size = 0
		return 0
	}
	diff := c.evictList.Length() - size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest(&evictedEntries)
	}
	c.size = size
	return diff
}

// Close destroys cleanup goroutine. To clean up the cache, run Purge() before Close().
func (c *LRU[K, V]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.done:
		return
	default:
	}
	close(c.done)
	c.cleanupStopped = true
}

// removeOldest removes the oldest item from the cache. Has to be called with lock!
func (c *LRU[K, V]) removeOldest(evicted *[]evictedEntry[K, V]) {
	if ent := c.evictList.Back(); ent != nil {
		c.removeElement(ent, evicted)
	}
}

// removeElement is used to remove a given list element from the cache. Has to be called with lock!
// The removed entry is recorded in evicted instead of invoking onEvict
// directly, so the callback can run after the lock is released.
func (c *LRU[K, V]) removeElement(e *internal.Entry[K, V], evicted *[]evictedEntry[K, V]) {
	c.evictList.Remove(e)
	delete(c.items, e.Key)
	c.removeFromBucket(e)
	if c.onEvict != nil {
		*evicted = append(*evicted, evictedEntry[K, V]{key: e.Key, value: e.Value})
	}
}

// deleteExpired deletes expired records from the oldest bucket, waiting for the newest entry
// in it to expire first.
func (c *LRU[K, V]) deleteExpired() {
	var evicted []evictedEntry[K, V]
	c.mu.Lock()

	// grab done channel to detect Closes
	done := c.done

	bucketIdx := c.nextCleanupBucket
	timeToExpire := time.Until(c.buckets[bucketIdx].newestEntry)
	// wait for newest entry to expire before cleanup without holding lock
	if timeToExpire > 0 {
		c.mu.Unlock()
		select {
		case <-time.After(timeToExpire):
		case <-done:
			return
		}
		c.mu.Lock()

		select {
		case <-done:
			// Done channel closed while sleeping, return without deleting entries
			c.mu.Unlock()
			return
		default:
		}
	}
	for _, ent := range c.buckets[bucketIdx].entries {
		c.removeElement(ent, &evicted)
	}
	c.nextCleanupBucket = (c.nextCleanupBucket + 1) % numBuckets
	c.mu.Unlock()
	c.fireCallbacks(evicted)
}

// addToBucket adds entry to expire bucket so that it will be cleaned up when the time comes. Has to be called with lock!
func (c *LRU[K, V]) addToBucket(e *internal.Entry[K, V]) {
	bucketID := (numBuckets + c.nextCleanupBucket - 1) % numBuckets
	e.ExpireBucket = bucketID
	c.buckets[bucketID].entries[e.Key] = e
	if c.buckets[bucketID].newestEntry.Before(e.ExpiresAt) {
		c.buckets[bucketID].newestEntry = e.ExpiresAt
	}
}

// removeFromBucket removes the entry from its corresponding bucket. Has to be called with lock!
func (c *LRU[K, V]) removeFromBucket(e *internal.Entry[K, V]) {
	delete(c.buckets[e.ExpireBucket].entries, e.Key)
}

// Cap returns the capacity of the cache
func (c *LRU[K, V]) Cap() int {
	return c.size
}

// Restart recreates the cleanup goroutine after Close() has been called.
func (c *LRU[K, V]) Restart() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if the goroutine is already running
	select {
	case <-c.done:
		// Channel is closed, need to recreate it
		c.done = make(chan struct{})
		c.cleanupStopped = false
		c.startGoroutine()
	default:
		// Goroutine is already running, nothing to do
	}
}

// startGoroutine starts the cleanup goroutine for expired entries.
func (c *LRU[K, V]) startGoroutine() {
	if c.ttl == noEvictionTTL {
		return
	}

	go func(done <-chan struct{}) {
		ticker := time.NewTicker(c.ttl / numBuckets)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				c.deleteExpired()
			}
		}
	}(c.done)
}
