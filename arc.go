package lru

import (
	"sync"

	"github.com/hashicorp/golang-lru/simplelru"
)

// ARCCache is a thread-safe fixed size Adaptive Replacement Cache (ARC).
// ARC is an enhancement over the standard LRU cache in that tracks both
// frequency and recency of use. This avoids a burst in access to new
// entries from evicting the frequently used older entries. It adds some
// additional tracking overhead to a standard LRU cache, computationally
// it is roughly 2x the cost, and the extra memory overhead is linear
// with the size of the cache. ARC has been patented by IBM, but is
// similar to the TwoQueueCache (2Q) which requires setting parameters.
type ARCCache struct {
	size int // Size is the total capacity of the cache
	p    int // P is the dynamic preference towards T1 or T2

	t1 simplelru.LRUCache // T1 is the LRU for recently accessed items
	b1 simplelru.LRUCache // B1 is the LRU for evictions from t1

	t2 simplelru.LRUCache // T2 is the LRU for frequently accessed items
	b2 simplelru.LRUCache // B2 is the LRU for evictions from t2

	lock RWLocker
}

// OptionARC defines option to customize ARCCache
type OptionARC func(*ARCCache) error

// NewARC creates an ARC of the given size
func NewARC(size int, opts ...OptionARC) (*ARCCache, error) {
	// Create the sub LRUs
	b1, err := simplelru.NewLRU(size, nil)
	if err != nil {
		return nil, err
	}
	b2, err := simplelru.NewLRU(size, nil)
	if err != nil {
		return nil, err
	}
	t1, err := simplelru.NewLRU(size, nil)
	if err != nil {
		return nil, err
	}
	t2, err := simplelru.NewLRU(size, nil)
	if err != nil {
		return nil, err
	}

	// Initialize the ARC
	c := &ARCCache{
		size: size,
		p:    0,
		t1:   t1,
		b1:   b1,
		t2:   t2,
		b2:   b2,
		lock: &sync.RWMutex{},
	}
	// Apply option settings
	for _, opt := range opts {
		if err = opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// NoLockARC disables locking for ARCCache
func NoLockARC(c *ARCCache) error {
	c.lock = NoOpRWLocker{}
	return nil
}

// Get looks up a key's value from the cache.
func (c *ARCCache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// If the value is contained in T1 (recent), then
	// promote it to T2 (frequent)
	if val, ok := c.t1.Peek(key); ok {
		c.t1.Remove(key)
		c.t2.Add(key, val)
		return val, ok
	}

	// Check if the value is contained in T2 (frequent)
	if val, ok := c.t2.Get(key); ok {
		return val, ok
	}

	// No hit
	return nil, false
}

// Add adds a value to the cache, return evicted key/val if it happens.
func (c *ARCCache) Add(key, value interface{}, evictedKeyVal ...*interface{}) (evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check if the value is contained in T1 (recent), and potentially
	// promote it to frequent T2
	if c.t1.Contains(key) {
		c.t1.Remove(key)
		c.t2.Add(key, value)
		return
	}

	// Check if the value is already in T2 (frequent) and update it
	if c.t2.Contains(key) {
		c.t2.Add(key, value)
		return
	}

	var evictedKey, evictedValue interface{}
	switch {
	case c.b1.Contains(key):
		// Check if this value was recently evicted as part of the
		// recently used list
		// T1 set is too small, increase P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b2Len > b1Len {
			delta = b2Len / b1Len
		}
		if c.p+delta >= c.size {
			c.p = c.size
		} else {
			c.p += delta
		}

		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.size {
			evictedKey, evictedValue, evicted = c.replace(false)
		}

		// Remove from B1
		c.b1.Remove(key)

		// Add the key to the frequently used list
		c.t2.Add(key, value)

	case c.b2.Contains(key):
		// Check if this value was recently evicted as part of the
		// frequently used list
		// T2 set is too small, decrease P appropriately
		delta := 1
		b1Len := c.b1.Len()
		b2Len := c.b2.Len()
		if b1Len > b2Len {
			delta = b1Len / b2Len
		}
		if delta >= c.p {
			c.p = 0
		} else {
			c.p -= delta
		}

		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.size {
			evictedKey, evictedValue, evicted = c.replace(true)
		}

		// Remove from B2
		c.b2.Remove(key)

		// Add the key to the frequently used list
		c.t2.Add(key, value)
	default:
		// Brand new entry
		// Potentially need to make room in the cache
		if c.t1.Len()+c.t2.Len() >= c.size {
			evictedKey, evictedValue, evicted = c.replace(false)
		}

		// Keep the size of the ghost buffers trim
		if c.b1.Len() > c.size-c.p {
			c.b1.RemoveOldest()
		}
		if c.b2.Len() > c.p {
			c.b2.RemoveOldest()
		}

		// Add to the recently seen list
		c.t1.Add(key, value)
	}
	if evicted && len(evictedKeyVal) > 0 {
		*evictedKeyVal[0] = evictedKey
	}
	if evicted && len(evictedKeyVal) > 1 {
		*evictedKeyVal[1] = evictedValue
	}
	return evicted
}

// replace is used to adaptively evict from either T1 or T2
// based on the current learned value of P
func (c *ARCCache) replace(b2ContainsKey bool) (k, v interface{}, ok bool) {
	t1Len := c.t1.Len()
	if t1Len > 0 && (t1Len > c.p || (t1Len == c.p && b2ContainsKey)) {
		k, v, ok = c.t1.RemoveOldest()
		if ok {
			c.b1.Add(k, nil)
		}
	} else {
		k, v, ok = c.t2.RemoveOldest()
		if ok {
			c.b2.Add(k, nil)
		}
	}
	return
}

// Len returns the number of cached entries
func (c *ARCCache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.t1.Len() + c.t2.Len()
}

// Keys returns all the cached keys
func (c *ARCCache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()
	k1 := c.t1.Keys()
	k2 := c.t2.Keys()
	return append(k1, k2...)
}

// Remove is used to purge a key from the cache
func (c *ARCCache) Remove(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.t1.Remove(key) {
		return true
	}
	if c.t2.Remove(key) {
		return true
	}
	if c.b1.Remove(key) {
		return false
	}
	if c.b2.Remove(key) {
		return false
	}
	return false
}

// Purge is used to clear the cache
func (c *ARCCache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.t1.Purge()
	c.t2.Purge()
	c.b1.Purge()
	c.b2.Purge()
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (c *ARCCache) Contains(key interface{}) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.t1.Contains(key) || c.t2.Contains(key)
}

// Peek is used to inspect the cache value of a key
// without updating recency or frequency.
func (c *ARCCache) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if val, ok := c.t1.Peek(key); ok {
		return val, ok
	}
	return c.t2.Peek(key)
}
