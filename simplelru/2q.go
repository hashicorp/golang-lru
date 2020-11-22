package simplelru

import (
	"fmt"
)

const (
	// Default2QRecentRatio is the ratio of the 2Q cache dedicated
	// to recently added entries that have only been accessed once.
	Default2QRecentRatio = 0.25

	// Default2QGhostEntries is the default ratio of ghost
	// entries kept to track entries recently evicted
	Default2QGhostEntries = 0.50
)

// TwoQueueLRU is a thread-safe fixed size 2Q LRU.
// 2Q is an enhancement over the standard LRU cache
// in that it tracks both frequently and recently used
// entries separately. This avoids a burst in access to new
// entries from evicting frequently used entries. It adds some
// additional tracking overhead to the standard LRU cache, and is
// computationally about 2x the cost, and adds some metadata over
// head. The ARCCache is similar, but does not require setting any
// parameters.
type TwoQueueLRU struct {
	size       int
	recentSize int

	recent      *LRU
	frequent    *LRU
	recentEvict *LRU
}

// New2Q creates a new TwoQueueLRU using the default
// values for the parameters.
func New2Q(size int) (*TwoQueueLRU, error) {
	return New2QParams(size, Default2QRecentRatio, Default2QGhostEntries)
}

// New2QParams creates a new TwoQueueLRU using the provided
// parameter values.
func New2QParams(size int, recentRatio, ghostRatio float64) (*TwoQueueLRU, error) {
	if size <= 0 {
		return nil, fmt.Errorf("invalid size")
	}
	if recentRatio < 0.0 || recentRatio > 1.0 {
		return nil, fmt.Errorf("invalid recent ratio")
	}
	if ghostRatio < 0.0 || ghostRatio > 1.0 {
		return nil, fmt.Errorf("invalid ghost ratio")
	}

	// Determine the sub-sizes
	recentSize := int(float64(size) * recentRatio)
	evictSize := int(float64(size) * ghostRatio)

	// Allocate the LRUs
	recent, err := NewLRU(size, nil)
	if err != nil {
		return nil, err
	}
	frequent, err := NewLRU(size, nil)
	if err != nil {
		return nil, err
	}
	recentEvict, err := NewLRU(evictSize, nil)
	if err != nil {
		return nil, err
	}

	// Initialize the cache
	c := &TwoQueueLRU{
		size:        size,
		recentSize:  recentSize,
		recent:      recent,
		frequent:    frequent,
		recentEvict: recentEvict,
	}
	return c, nil
}

// Get looks up a key's value from the cache.
func (c *TwoQueueLRU) Get(key interface{}) (value interface{}, ok bool) {
	// Check if this is a frequent value
	if val, ok := c.frequent.Get(key); ok {
		return val, ok
	}

	// If the value is contained in recent, then we
	// promote it to frequent
	if val, ok := c.recent.Peek(key); ok {
		c.recent.Remove(key)
		c.frequent.Add(key, val)
		return val, ok
	}

	// No hit
	return nil, false
}

// Add adds a value to the cache, return evicted key/val if eviction happens.
func (c *TwoQueueLRU) Add(key, value interface{}, evictedKeyVal ...*interface{}) (evicted bool) {
	// Check if the value is frequently used already,
	// and just update the value
	if c.frequent.Contains(key) {
		c.frequent.Add(key, value)
		return
	}

	// Check if the value is recently used, and promote
	// the value into the frequent list
	if c.recent.Contains(key) {
		c.recent.Remove(key)
		c.frequent.Add(key, value)
		return
	}

	var evictedKey, evictedValue interface{}
	// If the value was recently evicted, add it to the
	// frequently used list
	if c.recentEvict.Contains(key) {
		evictedKey, evictedValue, evicted = c.ensureSpace(true)
		c.recentEvict.Remove(key)
		c.frequent.Add(key, value)
	} else {
		// Add to the recently seen list
		evictedKey, evictedValue, evicted = c.ensureSpace(false)
		c.recent.Add(key, value)
	}
	if evicted && len(evictedKeyVal) > 0 {
		*evictedKeyVal[0] = evictedKey
	}
	if evicted && len(evictedKeyVal) > 1 {
		*evictedKeyVal[1] = evictedValue
	}
	return evicted
}

// ensureSpace is used to ensure we have space in the cache
func (c *TwoQueueLRU) ensureSpace(recentEvict bool) (key, value interface{}, evicted bool) {
	// If we have space, nothing to do
	recentLen := c.recent.Len()
	freqLen := c.frequent.Len()
	if recentLen+freqLen < c.size {
		return
	}

	// If the recent buffer is larger than
	// the target, evict from there
	if recentLen > 0 && (recentLen > c.recentSize || (recentLen == c.recentSize && !recentEvict)) {
		key, value, evicted = c.recent.RemoveOldest()
		c.recentEvict.Add(key, nil)
		return
	}

	// Remove from the frequent list otherwise
	return c.frequent.RemoveOldest()
}

// Len returns the number of items in the cache.
func (c *TwoQueueLRU) Len() int {
	return c.recent.Len() + c.frequent.Len()
}

// Keys returns a slice of the keys in the cache.
// The frequently used keys are first in the returned slice.
func (c *TwoQueueLRU) Keys() []interface{} {
	k1 := c.frequent.Keys()
	k2 := c.recent.Keys()
	return append(k1, k2...)
}

// Remove removes the provided key from the cache.
func (c *TwoQueueLRU) Remove(key interface{}) bool {
	if c.frequent.Remove(key) {
		return true
	}
	if c.recent.Remove(key) {
		return true
	}
	c.recentEvict.Remove(key)
	return false
}

// Purge is used to completely clear the cache.
func (c *TwoQueueLRU) Purge() {
	c.recent.Purge()
	c.frequent.Purge()
	c.recentEvict.Purge()
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (c *TwoQueueLRU) Contains(key interface{}) bool {
	return c.frequent.Contains(key) || c.recent.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (c *TwoQueueLRU) Peek(key interface{}) (value interface{}, ok bool) {
	if val, ok := c.frequent.Peek(key); ok {
		return val, ok
	}
	return c.recent.Peek(key)
}
