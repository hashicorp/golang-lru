package lru

import (
	"errors"
	"sync"
	"time"
)

type LruWithTTL struct {
	Cache
	schedule       map[interface{}]bool
	schedule_mutex sync.Mutex
}

// New creates an LRU of the given size
func NewTTL(size int) (*LruWithTTL, error) {
	return NewTTLWithEvict(size, nil)
}

func NewTTLWithEvict(size int, onEvicted func(key interface{}, value interface{})) (*LruWithTTL, error) {
	if size <= 0 {
		return nil, errors.New("Must provide a positive size")
	}
	c, _ := NewWithEvict(size, onEvicted)
	return &LruWithTTL{*c, make(map[interface{}]bool), sync.Mutex{}}, nil
}

func (this *LruWithTTL) clearSchedule(key interface{}) {
	this.schedule_mutex.Lock()
	defer this.schedule_mutex.Unlock()
	delete(this.schedule, key)
}

func (this *LruWithTTL) AddWithTTL(key, value interface{}, ttl time.Duration) bool {
	this.schedule_mutex.Lock()
	defer this.schedule_mutex.Unlock()
	if this.schedule[key] {
		// already scheduled, nothing to do
	} else {
		this.schedule[key] = true
		// Schedule cleanup
		go func() {
			defer this.Cache.Remove(key)
			defer this.clearSchedule(key)
			time.Sleep(ttl)
		}()
	}
	return this.Cache.Add(key, value)
}
