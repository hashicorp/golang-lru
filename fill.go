// This package provides a simple LRU cache. It is based on the
// LRU implementation in groupcache:
// https://github.com/golang/groupcache/tree/master/lru
package lru

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

type FillFunc func() (interface{}, time.Time)

// Cache is a thread-safe fixed size LRU cache.
type FillingCache struct {
	size      int
	evictList *list.List
	items     map[interface{}]*list.Element
	lock      sync.RWMutex

	fill func(key interface{}) (interface{}, time.Time, error)
}

// entry is used to hold a value in the evictList
type exp_entry struct {
	key   interface{}
	value interface{}
	exp   time.Time
	err   error
	wg    sync.WaitGroup
}

// New creates an LRU of the given size
func NewFilling(size int) (*FillingCache, error) {
	if size <= 0 {
		return nil, errors.New("Must provide a positive size")
	}
	c := &FillingCache{
		size:      size,
		evictList: list.New(),
		items:     make(map[interface{}]*list.Element, size),
	}
	return c, nil
}

// looks up a key's value from the cache, if it is not present get it
func (c *FillingCache) Get(key interface{}) (value interface{}, err error) {
	c.lock.Lock()

	entry, ok := c.items[key]
	if ok {
		c.lock.Unlock()
		casted := entry.Value.(*exp_entry)
		casted.wg.Wait()

		if time.Now().Before(casted.exp) {
			c.evictList.MoveToFront(entry)
			return casted.value, casted.err
		}

		//in the case where multiple clients are waiting,
		//and then they receive an expired object
		//they will now thunder (They will all try to get the value)
		//thats a really unlikely failure
		c.lock.Lock()
	}

	casted := &exp_entry{key: key}
	casted.wg.Add(1) //stop clients from returning with stale data

	entry = c.evictList.PushFront(casted)
	c.items[key] = entry

	//free up the map while we fetch our object
	c.lock.Unlock()
	casted.value, casted.exp, casted.err = c.fill(key)
	casted.wg.Done()

	//re-lock to trim set
	c.lock.Lock()
	if c.evictList.Len() > c.size {
		c.removeOldest()
	}
	c.lock.Unlock()

	return casted.value, casted.err
}

// removeOldest removes the oldest item from the cache.
func (c *FillingCache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *FillingCache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*exp_entry)
	delete(c.items, kv.key)
}
