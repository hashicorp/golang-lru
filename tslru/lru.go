// Package tslru provides thread safe LRU implementation based on build-in channel.
package tslru

import (
	"container/list"
	"errors"
	"sync"
	"sync/atomic"
)

type actionType string

const (
	hitAction    actionType = "hit" // Get()
	addAction    actionType = "add"
	delAction    actionType = "del"
	oldestAction actionType = "get_old"
	iterAction   actionType = "iter"
	purgeAction  actionType = "purge"
)

type action struct {
	t   actionType
	ele *entry

	o1 chan interface{}
	o2 chan *entry
	o3 chan struct{}
}

// LRU implements a thread safe fixed size LRU cache
type LRU struct {
	mu        sync.RWMutex
	limit     int32
	evictList *list.List
	items     map[interface{}]*entry
	ctl       chan action
}

// entry is used to hold a value in the evictList
type entry struct {
	key     interface{}
	value   interface{}
	element *list.Element
}

// NewLRU constructs an LRU of the given size
func NewLRU(size int) (*LRU, error) {
	if size <= 0 {
		return nil, errors.New("must provide a positive size")
	}
	c := &LRU{
		limit:     int32(size),
		evictList: list.New(),
		ctl:       make(chan action, 1024),
		items:     make(map[interface{}]*entry),
	}
	go c.work()
	return c, nil
}

func (c *LRU) work() {
	defer close(c.ctl)
	for {
		act := <-c.ctl
		switch act.t {
		case hitAction:
			if ele := act.ele.element; ele != nil {
				c.evictList.MoveToFront(ele)
			}
		case addAction:
			limit := int(atomic.LoadInt32(&c.limit))
			c.mu.Lock()
			if len(c.items) > limit+500 {
				ele := c.evictList.Back()
				for len(c.items) > limit && ele != nil {
					delete(c.items, ele.Value.(*entry).key)
					ele2 := ele.Prev()
					c.evictList.Remove(ele)
					ele = ele2
				}
			}
			c.mu.Unlock()
			act.ele.element = c.evictList.PushFront(act.ele)
		case delAction:
			c.evictList.Remove(act.ele.element)
		case oldestAction:
			if ele := c.evictList.Back(); ele != nil {
				act.o2 <- ele.Value.(*entry)
			}
		case iterAction:
			for ele := c.evictList.Back(); ele != nil && ele != c.evictList.Front(); ele = ele.Prev() {
				act.o1 <- ele.Value.(*entry).key
			}
			close(act.o1)
		case purgeAction:
			c.evictList.Init()
			act.o3 <- struct{}{}
		}
	}
}

// Len Returns the number of items in the items.
func (c *LRU) Len() int {
	c.mu.RLock()
	size := len(c.items)
	c.mu.RUnlock()
	return size
}

// Add adds a value to the items.  Returns true if an eviction occurred.
func (c *LRU) Add(key, value interface{}) (evicted bool) {
	ent := &entry{key: key, value: value}
	c.mu.Lock()
	c.items[key] = ent
	evicted = len(c.items) > int(atomic.LoadInt32(&c.limit))
	c.mu.Unlock()
	c.ctl <- action{t: addAction, ele: ent}
	return
}

// Get looks up a key's value from the cache.
func (c *LRU) Get(key interface{}) (value interface{}, ok bool) {
	c.mu.RLock()
	ent, ok := c.items[key]
	c.mu.RUnlock()
	if ok {
		select {
		case c.ctl <- action{t: hitAction, ele: ent}:
		default:
			// log
		}
		return ent.value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness
// or deleting it for being stale.
func (c *LRU) Contains(key interface{}) (ok bool) {
	c.mu.RLock()
	_, ok = c.items[key]
	c.mu.RUnlock()
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU) Peek(key interface{}) (value interface{}, ok bool) {
	c.mu.RLock()
	ent, ok := c.items[key]
	c.mu.RUnlock()
	if ok {
		return ent.value, true
	}
	return
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU) Remove(key interface{}) (present bool) {
	c.mu.Lock()
	ent, ok := c.items[key]
	if ok {
		delete(c.items, key)
		c.mu.Unlock()
		c.ctl <- action{t: delAction, ele: ent}
		return true
	}
	c.mu.Unlock()
	return false
}

// RemoveOldest Removes the oldest entry from cache.
func (c *LRU) RemoveOldest() (interface{}, interface{}, bool) {
	if k, v, ok := c.GetOldest(); ok {
		c.Remove(k)
		return k, v, true
	}
	return nil, nil, false
}

// GetOldest Returns the oldest entry from the cache. #key, value, isFound
func (c *LRU) GetOldest() (interface{}, interface{}, bool) {
	a := action{t: oldestAction, o2: make(chan *entry)}
	c.ctl <- a

	if ent := <-a.o2; ent != nil {
		return ent.key, ent.value, true
	}
	return nil, nil, false
}

// Keys Returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU) Keys() []interface{} {
	a := action{t: iterAction, o1: make(chan interface{})}
	c.ctl <- a

	var ret = make([]interface{}, 0, c.Len())
	for k := range a.o1 {
		ret = append(ret, k)
	}
	return ret
}

// Purge Clears all cache entries.
func (c *LRU) Purge() {
	c.mu.Lock()
	c.items = make(map[interface{}]*entry)
	c.mu.Unlock()

	a := action{t: purgeAction, o3: make(chan struct{})}
	c.ctl <- a
	<-a.o3
}

// Resize cache, returning number evicted
func (c *LRU) Resize(newSize int) int {
	diff := newSize - int(atomic.LoadInt32(&c.limit))
	atomic.StoreInt32(&c.limit, int32(newSize))

	num := 0
	for diff < 0 {
		if _, _, ok := c.RemoveOldest(); !ok {
			break
		}
		num++
		diff++
	}
	return num
}
