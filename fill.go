// This package provides a simple LRU cache. It is based on the
// LRU implementation in groupcache:
// https://github.com/golang/groupcache/tree/master/lru
package lru

import (
	"container/list"
	"sync"
	"time"
)

type Filler interface {
	// Fill returns the value and expiration time of the given key.
	// When Fill fails to get the key, it can return a non-zero expiration to cache the error entry.
	Fill(key interface{}) (value interface{}, expiration time.Time, err error)
}

type FillFunc func(key interface{}) (value interface{}, expiration time.Time, err error)

func (ff FillFunc) Fill(key interface{}) (value interface{}, expiration time.Time, err error) {
	return ff(key)
}

// FillingCache is a LRU cache. It fills an entry the first time the entry is
// retrieved.  It ensures the entry is filled once even when the entry is
// retrieved by multiple clients.  An entry may expire. An expired entry goes
// through the filling process the same as a new entry.
// Note that if Fill fails, the error entry is cached.
type FillingCache struct {
	size      int
	evictList *list.List
	items     map[interface{}]*list.Element
	lock      sync.RWMutex
	filler    Filler
}

// expirableEntry holds a value in the evictList
type expirableEntry struct {
	key   interface{}
	value interface{}
	exp   time.Time
	err   error
	wg    sync.WaitGroup
}

// NewFillingCache returns a FillingCache of the given size.
func NewFillingCache(size int, fr Filler) *FillingCache {
	if size <= 0 {
		panic("invalid cache size")
	}
	return &FillingCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element, size),
		filler:    fr,
	}
}

// looks up a key's value from the cache, if it is not present get it
func (c *FillingCache) Get(key interface{}) (value interface{}, err error) {
	entry, tofill := c.getEntry(key)

	if tofill {
		value, exp, err := c.filler.Fill(key)
		if exp.IsZero() {
			// zero expiration means it is already expired.
			// The entry will be removed when the key is accessed the next time.
			exp = time.Now().Add(-time.Hour)
		}

		c.lock.Lock()
		entry.value = value
		if !entry.exp.IsZero() {
			panic("BUG: entry.exp is set while it is being filled")
		}
		entry.exp = exp
		entry.err = err
		c.lock.Unlock()
		entry.wg.Done() // signal other clients
		return value, err
	}

	// always access entry after it is filled
	entry.wg.Wait()
	return entry.value, entry.err
}

func (c *FillingCache) getEntry(key interface{}) (entry *expirableEntry, tofill bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	item, ok := c.items[key]
	if ok {
		entry = item.Value.(*expirableEntry)
		c.evictList.Remove(item)
	}
	if !ok || (!entry.exp.IsZero() && entry.exp.Before(time.Now())) {
		// if the entry is not found or already expired, add a new entry.
		// the first client that adds the entry fills the entry
		entry = &expirableEntry{key: key}
		entry.wg.Add(1)
		tofill = true
	}
	// either add the new entry or move an existing entry to the front
	c.items[key] = c.evictList.PushFront(entry)

	// TODO: remove the expired entries instead of the oldest one.
	if c.evictList.Len() > c.size {
		c.removeOldest()
	}
	return entry, tofill
}

// removeOldest removes the oldest item from the cache.
func (c *FillingCache) removeOldest() {
	item := c.evictList.Back()
	if item == nil {
		return
	}
	c.evictList.Remove(item)
	entry := item.Value.(*expirableEntry)
	delete(c.items, entry.key)
}
