package lru

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/golang-lru/simplelru"
)

// common interface shared by 2q, arc and simple LRU, used as interface of backing LRU
type lruCache interface {
	// Adds a value to the cache, returns evicted <k,v> if happened and
	// updates the "recently used"-ness of the key.
	Add(k, v interface{}, evictedKeyVal ...*interface{}) (evicted bool)
	// Returns key's value from the cache if found and
	// updates the "recently used"-ness of the key.
	Get(k interface{}) (v interface{}, ok bool)
	// Removes a key from the cache
	Remove(k interface{}) bool
	// Returns key's value without updating the "recently used"-ness of the key.
	Peek(key interface{}) (value interface{}, ok bool)
	// Checks if a key exists in cache without updating the recent-ness.
	Contains(k interface{}) bool
	// Returns a slice of the keys in the cache, from oldest to newest.
	Keys() []interface{}
	// Returns the number of items in the cache.
	Len() int
	// Clears all cache entries.
	Purge()
}

type entry struct {
	key            interface{}
	val            interface{}
	expirationTime time.Time
	elem           *list.Element
}

func (e entry) String() string {
	return fmt.Sprintf("%v,%v  %v", e.key, e.val, e.expirationTime)
}

// two expiration policies
type expiringType byte

const (
	expireAfterWrite expiringType = iota
	expireAfterAccess
)

// ExpiringCache will wrap an existing LRU and make its entries expiring
// according to two policies:
// expireAfterAccess and expireAfterWrite (default)
// Internally keep a expireList sorted by entries' expirationTime
type ExpiringCache struct {
	lru        lruCache
	expiration time.Duration
	expireList *expireList
	expireType expiringType
	// placeholder for time.Now() for easier testing setup
	timeNow func() time.Time
	lock    sync.RWMutex
}

// OptionExp defines option to customize ExpiringCache
type OptionExp func(c *ExpiringCache) error

// NewExpiring2Q creates an expiring cache with specifized
// size and entries lifetime duration, backed by a 2-queue LRU
func NewExpiring2Q(size int, expir time.Duration, opts ...OptionExp) (elru *ExpiringCache, err error) {
	lru, err := simplelru.New2Q(size)
	if err != nil {
		return
	}
	elru, err = Expiring(expir, lru, opts...)
	return
}

// NewExpiringARC creates an expiring cache with specifized
// size and entries lifetime duration, backed by a ARC LRU
func NewExpiringARC(size int, expir time.Duration, opts ...OptionExp) (elru *ExpiringCache, err error) {
	lru, err := simplelru.NewARC(size)
	if err != nil {
		return
	}
	elru, err = Expiring(expir, lru, opts...)
	return
}

// NewExpiringLRU creates an expiring cache with specifized
// size and entries lifetime duration, backed by a simple LRU
func NewExpiringLRU(size int, expir time.Duration, opts ...OptionExp) (elru *ExpiringCache, err error) {
	lru, err := simplelru.NewLRU(size, nil)
	if err != nil {
		return
	}
	elru, err = Expiring(expir, lru, opts...)
	return
}

// Expiring will wrap an existing LRU to make its entries
// expiring with specified duration
func Expiring(expir time.Duration, lru lruCache, opts ...OptionExp) (*ExpiringCache, error) {
	// create expiring cache with default settings
	elru := &ExpiringCache{
		lru:        lru,
		expiration: expir,
		expireList: newExpireList(),
		expireType: expireAfterWrite,
		timeNow:    time.Now,
	}
	// apply options to customize
	for _, opt := range opts {
		if err := opt(elru); err != nil {
			return nil, err
		}
	}
	return elru, nil
}

// ExpireAfterWrite sets expiring policy
func ExpireAfterWrite(elru *ExpiringCache) error {
	elru.expireType = expireAfterWrite
	return nil
}

// ExpireAfterAccess sets expiring policy
func ExpireAfterAccess(elru *ExpiringCache) error {
	elru.expireType = expireAfterAccess
	return nil
}

// TimeTicker sets the function used to return current time, for test setup
func TimeTicker(tn func() time.Time) OptionExp {
	return func(elru *ExpiringCache) error {
		elru.timeNow = tn
		return nil
	}
}

// Add add a key/val pair to cache with cache's default expiration duration
// return evicted key/val pair if eviction happens.
// Should be used in most cases for better performance
func (elru *ExpiringCache) Add(k, v interface{}, evictedKeyVal ...*interface{}) (evicted bool) {
	return elru.AddWithTTL(k, v, elru.expiration, evictedKeyVal...)
}

// AddWithTTL add a key/val pair to cache with provided expiration duration
// return evicted key/val pair if eviction happens.
// Using this with variant expiration durations could cause degraded performance
func (elru *ExpiringCache) AddWithTTL(k, v interface{}, expiration time.Duration, evictedKeyVal ...*interface{}) (evicted bool) {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	now := elru.timeNow()
	var ent *entry
	var expired []*entry
	if ent0, _ := elru.lru.Peek(k); ent0 != nil {
		// update existing cache entry
		ent = ent0.(*entry)
		ent.val = v
		ent.expirationTime = now.Add(expiration)
		elru.expireList.MoveToFront(ent)
	} else {
		// first remove 1 possible expiration to add space for new entry
		expired = elru.removeExpired(now, false)
		// add new entry to expiration list
		ent = &entry{
			key:            k,
			val:            v,
			expirationTime: now.Add(expiration),
		}
		elru.expireList.PushFront(ent)
	}
	// Add/Update cache entry in backing cache
	var evictedKey, evictedVal interface{}
	evicted = elru.lru.Add(k, ent, &evictedKey, &evictedVal)
	// remove evicted ent from expireList
	if evicted {
		ent = evictedVal.(*entry)
		evictedVal = ent.val
		elru.expireList.Remove(ent)
	} else if len(expired) > 0 {
		evictedKey = expired[0].key
		evictedVal = expired[0].val
		evicted = true
	}
	if evicted && len(evictedKeyVal) > 0 {
		*evictedKeyVal[0] = evictedKey
	}
	if evicted && len(evictedKeyVal) > 1 {
		*evictedKeyVal[1] = evictedVal
	}
	return evicted
}

// Get returns key's value from the cache if found
func (elru *ExpiringCache) Get(k interface{}) (v interface{}, ok bool) {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	now := elru.timeNow()
	if ent0, ok := elru.lru.Get(k); ok {
		ent := ent0.(*entry)
		if ent.expirationTime.After(now) {
			if elru.expireType == expireAfterAccess {
				ent.expirationTime = now.Add(elru.expiration)
				elru.expireList.MoveToFront(ent)
			}
			return ent.val, true
		}
	}
	return
}

// Remove removes a key from the cache
func (elru *ExpiringCache) Remove(k interface{}) bool {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	if ent, _ := elru.lru.Peek(k); ent != nil {
		elru.expireList.Remove(ent.(*entry))
		return elru.lru.Remove(k)
	}
	return false
}

// Peek return key's value without updating the "recently used"-ness of the key.
// returns ok=false if k not found or entry expired
func (elru *ExpiringCache) Peek(k interface{}) (v interface{}, ok bool) {
	elru.lock.RLock()
	defer elru.lock.RUnlock()
	if ent0, ok := elru.lru.Peek(k); ok {
		ent := ent0.(*entry)
		if ent.expirationTime.After(elru.timeNow()) {
			return ent.val, true
		}
		return ent.val, false
	}
	return
}

// Contains is used to check if the cache contains a key
// without updating recency or frequency.
func (elru *ExpiringCache) Contains(k interface{}) bool {
	_, ok := elru.Peek(k)
	return ok
}

// Keys returns a slice of the keys in the cache.
// The frequently used keys are first in the returned slice.
func (elru *ExpiringCache) Keys() []interface{} {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	// to get accurate key set, remove all expired
	elru.removeExpired(elru.timeNow(), true)
	return elru.lru.Keys()
}

// Len returns the number of items in the cache.
func (elru *ExpiringCache) Len() int {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	// to get accurate size, remove all expired
	elru.removeExpired(elru.timeNow(), true)
	return elru.lru.Len()
}

// Purge is used to completely clear the cache.
func (elru *ExpiringCache) Purge() {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	elru.expireList.Init()
	elru.lru.Purge()
}

// RemoveAllExpired remove all expired entries, can be called by cleanup goroutine
func (elru *ExpiringCache) RemoveAllExpired() {
	elru.lock.Lock()
	defer elru.lock.Unlock()
	elru.removeExpired(elru.timeNow(), true)
}

// either remove one (the oldest expired), or all expired
func (elru *ExpiringCache) removeExpired(now time.Time, removeAllExpired bool) (res []*entry) {
	res = elru.expireList.RemoveExpired(now, removeAllExpired)
	for i := 0; i < len(res); i++ {
		elru.lru.Remove(res[i].key)
	}
	return
}

// oldest entries are at front of expire list
type expireList struct {
	expList *list.List
}

func newExpireList() *expireList {
	return &expireList{
		expList: list.New(),
	}
}

func (el *expireList) Init() {
	el.expList.Init()
}

func (el *expireList) PushFront(ent *entry) {
	// When all operations use ExpiringCache default expiration,
	// PushFront should succeed at first/front entry of list
	for e := el.expList.Front(); e != nil; e = e.Next() {
		if !ent.expirationTime.Before(e.Value.(*entry).expirationTime) {
			ent.elem = el.expList.InsertBefore(ent, e)
			return
		}
	}
	ent.elem = el.expList.PushBack(ent)
}

func (el *expireList) MoveToFront(ent *entry) {
	// When all operations use ExpiringCache default expiration,
	// MoveToFront should succeed at first/front entry of list
	for e := el.expList.Front(); e != nil; e = e.Next() {
		if !ent.expirationTime.Before(e.Value.(*entry).expirationTime) {
			el.expList.MoveBefore(ent.elem, e)
			return
		}
	}
	el.expList.MoveAfter(ent.elem, el.expList.Back())
}

func (el *expireList) Remove(ent *entry) interface{} {
	return el.expList.Remove(ent.elem)
}

// either remove one (the oldest expired), or remove all expired
func (el *expireList) RemoveExpired(now time.Time, removeAllExpired bool) (res []*entry) {
	for {
		back := el.expList.Back()
		if back == nil || back.Value.(*entry).expirationTime.After(now) {
			break
		}
		// expired
		ent := el.expList.Remove(back).(*entry)
		res = append(res, ent)
		if !removeAllExpired {
			break
		}
	}
	return
}
