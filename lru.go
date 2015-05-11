// This package provides a simple LRU cache. It is based on the
// LRU implementation in groupcache:
// https://github.com/golang/groupcache/tree/master/lru
package lru

import (
    "container/list"
    "errors"
    "sync"
    "time"
    "encoding/gob"
    "bytes"
    "log"
)

// Cache is a thread-safe fixed size LRU cache.
type Cache struct {
    maxBytes    int64
    bytes       int64
    maxItems    int
    evictList   *list.List
    items       map[interface{}]*list.Element
    lock        sync.Mutex
    onRemove    func(interface{})
}

// entry is used to hold a value in the evictList
type entry struct {
    Key   interface{}
    Value interface{}
}

// New creates an LRU cache. It accepts a maximum size limit
// and a maximum byte limit. If either one of these are set 
// to zero, then that limit will be considered unlimitted but 
// if both are set to zero then an error will be returned.
// Exceeding either of these limits while calling Add() will
// result in the least recently used item being dropped from
// the cache and the onRemove() function being called for
// that item.
func New(items int, bytes int64) (*Cache, error) {
    if items <= 0 && bytes <= int64(0) {
        return nil, errors.New("Must provide a positive bound on either item count or byte count.")
    }
    onRemove := func(val interface{}) {
        return
    }
    c := &Cache{
        maxBytes:   bytes,
        bytes:      int64(0),
        maxItems:   items,
        evictList:  list.New(),
        items:      make(map[interface{}]*list.Element),
        onRemove:   onRemove,
    }
    return c, nil
}

// Purge will completely clear the cache of all key, value pairs.
func (c *Cache) Purge() {
    c.lock.Lock()
    defer c.lock.Unlock()            
    for c.Len() > 0 {
        c.removeOldest()
    }
    c.items = nil
    c.items = make(map[interface{}]*list.Element)
}

// Schedule routine purges. May be used to free up memory or 
// ensure that the OnRemove function is called within a certain
// amount of time.
func (cache *Cache) ScheduleClear(d time.Duration) {
    t := time.Tick(d)
    for {
        select {
        case <-t:
            cache.Purge()
        }
    }
}

func getByteCount(key interface{}) int64 {
    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    err := enc.Encode(key)
    if err != nil {
    	log.Println(err)
        return int64(0)
    }
    return int64(len(buf.Bytes()))
}

// Add adds a value to the cache. If the new value causes the
// set maxItems or maxBytes values to be exceeded, then the
// least recently used item will be popped from the cache and 
// the onRemove() function will be called for that value.
func (c *Cache) Add(key, value interface{}) {
    c.lock.Lock()
    defer c.lock.Unlock()

    // Check for existing item
    if ent, ok := c.items[key]; ok {
        c.evictList.MoveToFront(ent)
        ent.Value.(*entry).Value = value
        return
    }

    // Add new item
    ent := &entry{key, value}
    entry := c.evictList.PushFront(ent)
    c.items[key] = entry
    c.bytes += getByteCount(value)

    // Verify size not exceeded
    if (c.maxItems > 0 && c.evictList.Len() > c.maxItems) || (c.maxBytes > 0 && c.bytes > c.maxBytes) {
        c.removeOldest()
    }
}

// Allows for making changes to a value in the cache.
// Returns a boolean indicating whether the key was found.
func (c *Cache) Update(key interface{}, f func(val interface{})) bool {
    ent, ok := c.items[key]
    if !ok {
        return false
    }
    f(ent.Value.(*entry).Value)
    entry := c.evictList.PushFront(ent)
    c.items[key] = entry
    return true
}

// UpdateWithoutReorder() works like Update()
// without pushing it back to the front of the list. 
// Maintains order of Least Recently Used.
func (c *Cache) UpdateWithoutReorder(key interface{}, f func(val interface{})) bool {
    ent, ok := c.items[key]
    if !ok {
        return false
    }
    f(ent.Value.(*entry).Value)
    return true
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
    c.lock.Lock()
    defer c.lock.Unlock()

    if ent, ok := c.items[key]; ok {
        c.evictList.MoveToFront(ent)
        return ent.Value.(*entry).Value, true
    }
    return
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key interface{}) {
    c.lock.Lock()
    defer c.lock.Unlock()

    if ent, ok := c.items[key]; ok {
        c.removeElement(ent)
    }
}

// Specify a function to run before deleting an item from the cache.
// Function will be passed the value currently stored in interface{}
// associated with the key removed.
func (c *Cache) OnRemove(f func(interface{})) {
    c.onRemove = f
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
    c.lock.Lock()
    defer c.lock.Unlock()
    c.removeOldest()
}

// removeOldest removes the oldest item from the cache.
func (c *Cache) removeOldest() {
    ent := c.evictList.Back()
    if ent != nil {
        c.removeElement(ent)
        c.onRemove(ent.Value.(*entry).Value)
    }
}

// removeElement is used to remove a given list element from the cache.
func (c *Cache) removeElement(e *list.Element) {
    c.evictList.Remove(e)
    kv := e.Value.(*entry)
    c.bytes -= getByteCount(e.Value)
    c.items[kv.Key] = nil  // force garbage collection
    delete(c.items, kv.Key)
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
    c.lock.Lock()
    defer c.lock.Unlock()
    return c.evictList.Len()
}
