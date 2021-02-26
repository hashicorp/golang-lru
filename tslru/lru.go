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
	limit     int32
	size      int32
	evictList *list.List
	items     sync.Map
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
			c.evictList.MoveToFront(act.ele.element)
		case addAction:
			c.items.Store(act.ele.key, act.ele)
			atomic.AddInt32(&c.size, 1)

			act.o <- c.gc() > 0
			act.ele.element = c.evictList.PushFront(act.ele)
		case delAction:
			_, ok := c.items.Load(act.ele.key)
			c.items.Delete(act.ele.key)
			atomic.AddInt32(&c.size, -1)

			act.o <- ok
			c.evictList.Remove(act.ele.element)
		case oldestAction:
			act.o <- c.evictList.Back().Value.(*entry)
		case iterAction:
			for ele := c.evictList.Back(); ele != nil && ele != c.evictList.Front(); ele = ele.Prev() {
				act.o <- ele.Value.(*entry).key
			}
			close(act.o)
		case purgeAction:
			c.items.Range(func(key, value interface{}) bool {
				c.items.Delete(key)
				atomic.AddInt32(&c.size, -1)
				return true
			})

			c.evictList.Init()
			act.o <- struct{}{}
		case gcAction:
			n := c.gc()
			act.o <- n
		}
	}
}

func (c *LRU) gc() (n int) {
	ele := c.evictList.Back()

	for atomic.LoadInt32(&c.size) > atomic.LoadInt32(&c.limit) && ele != nil {
		c.items.Delete(ele.Value.(*entry).key)
		atomic.AddInt32(&c.size, -1)
		ele2 := ele.Prev()
		c.evictList.Remove(ele)
		ele = ele2
		n++
	}
	return
}

// Len Returns the number of items in the items.
func (c *LRU) Len() int {
	return int(atomic.LoadInt32(&c.size))
}

// Add adds a value to the items.  Returns true if an eviction occurred.
func (c *LRU) Add(key, value interface{}) (evicted bool) {
	itf, ok := c.items.Load(key)
	if ok {
		if ent := itf.(*entry); ent.value == value {
			c.ctl <- action{t: hitAction, ele: ent}
			return false
		}
		c.Remove(key)
	}

	ent := &entry{key: key, value: value}
	a := action{t: addAction, ele: ent, o: make(chan interface{})}
	c.ctl <- a

	b := <-a.o
	return b.(bool)
}

// Get looks up a key's value from the cache.
func (c *LRU) Get(key interface{}) (value interface{}, ok bool) {
	itf, ok := c.items.Load(key)
	if ok {
		ent := itf.(*entry)
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
	_, ok = c.items.Load(key)
	return ok
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *LRU) Peek(key interface{}) (value interface{}, ok bool) {
	itf, ok := c.items.Load(key)
	if ok {
		return itf.(*entry).value, true
	}
	return
}

// Remove removes the provided key from the cache, returning if the
// key was contained.
func (c *LRU) Remove(key interface{}) (present bool) {
	itf, ok := c.items.Load(key)
	if ok {
		a := action{t: delAction, ele: itf.(*entry), o: make(chan interface{})}
		c.ctl <- a

		b := <-a.o
		return b.(bool)
	}
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
