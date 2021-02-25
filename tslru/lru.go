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
	hitAction    actionType = "hit"
	addAction    actionType = "add"
	delAction    actionType = "del"
	oldestAction actionType = "get_old"
	iterAction   actionType = "iter"
	purgeAction  actionType = "purge"
	gcAction     actionType = "gc"
)

type action struct {
	t   actionType
	ele *entry

	o chan interface{}
}

// LRU implements a thread safe fixed size LRU cache
type LRU struct {
	mu        sync.RWMutex
	limit     int32
	evictList *list.List
	items     map[interface{}]*entry
	ctl       chan action
	pool      sync.Pool
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
		pool: sync.Pool{New: func() interface{} {
			return &entry{}
		}},
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
			c.mu.RLock()
			_, ok := c.items[act.ele.key]
			c.mu.RUnlock()
			if ok {
				c.evictList.MoveToFront(act.ele.element)
			}
		case addAction:
			c.mu.Lock()
			c.items[act.ele.key] = act.ele
			act.o <- c.gc() > 0
			c.mu.Unlock()
			act.ele.element = c.evictList.PushFront(act.ele)
		case delAction:
			c.evictList.Remove(act.ele.element)
		case oldestAction:
			act.o <- c.evictList.Back().Value.(*entry)
		case iterAction:
			for ele := c.evictList.Back(); ele != nil && ele != c.evictList.Front(); ele = ele.Prev() {
				act.o <- ele.Value.(*entry).key
			}
			close(act.o)
		case purgeAction:
			c.mu.Lock()
			c.items = make(map[interface{}]*entry)
			c.mu.Unlock()

			c.evictList.Init()
			act.o <- struct{}{}
		case gcAction:
			c.mu.Lock()
			n := c.gc()
			c.mu.Unlock()
			act.o <- n
		}
	}
}

func (c *LRU) gc() (n int) {
	ele := c.evictList.Back()
	for len(c.items) > int(atomic.LoadInt32(&c.limit)) && ele != nil {
		n++
		delete(c.items, ele.Value.(*entry).key)
		ele2 := ele.Prev()
		v := c.evictList.Remove(ele)
		c.pool.Put(v)
		ele = ele2
	}
	return
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
	c.mu.RLock()
	ent, ok := c.items[key]
	c.mu.RUnlock()
	if ok {
		c.ctl <- action{t: hitAction, ele: ent}
		return false
	}

	ent = c.pool.Get().(*entry)
	ent.key = key
	ent.value = value
	a := action{t: addAction, ele: ent, o: make(chan interface{})}
	c.ctl <- a

	b := <-a.o
	return b.(bool)
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
	a := action{t: oldestAction, o: make(chan interface{})}
	c.ctl <- a

	if o := <-a.o; o != nil {
		ent := o.(*entry)
		return ent.key, ent.value, true
	}
	return nil, nil, false
}

// Keys Returns a slice of the keys in the cache, from oldest to newest.
func (c *LRU) Keys() []interface{} {
	a := action{t: iterAction, o: make(chan interface{})}
	c.ctl <- a

	var ret = make([]interface{}, 0, c.Len())
	for k := range a.o {
		ret = append(ret, k)
	}
	return ret
}

// Purge Clears all cache entries.
func (c *LRU) Purge() {
	a := action{t: purgeAction, o: make(chan interface{})}
	c.ctl <- a
	<-a.o
}

// Resize cache, returning number evicted
func (c *LRU) Resize(newSize int) int {
	atomic.StoreInt32(&c.limit, int32(newSize))
	a := action{t: gcAction, o: make(chan interface{})}
	c.ctl <- a

	b := <-a.o
	return b.(int)
}
