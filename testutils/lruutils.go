package testutils

import (
	"testing"

	"github.com/hashicorp/golang-lru/simplelru"
)

func BasicTest(t *testing.T, l simplelru.LRUCache, capacity int, evictCounter *int) {
	// add twice as much the capacity to check if eviction occurs
	for i := 0; i < 2*capacity; i++ {
		l.Add(i, i)
	}

	if l.Len() != capacity {
		t.Fatalf("bad len: %v", l.Len())
	}

	// half of them should be evicted to make room for the incoming ones
	if *evictCounter != capacity {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	// cache should contain only the keys from capacity..2*capacity, anything before
	// that should have been evicted
	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+capacity {
			t.Fatalf("bad key: %v", k)
		}
	}

	for i := 0; i < capacity; i++ {
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be evicted")
		}
	}

	for i := capacity; i < 2*capacity; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}

	// delete half the items from cache
	lastIndex := (capacity + capacity/2)
	for i := capacity; i < lastIndex; i++ {
		ok := l.Remove(i)
		if !ok {
			t.Fatalf("should be contained")
		}
		ok = l.Remove(i)
		if ok {
			t.Fatalf("should not be contained")
		}
		_, ok = l.Get(i)
		if ok {
			t.Fatalf("should be deleted")
		}
	}

	// this makes this item the most recently accessed; moved to the front
	l.Get(lastIndex) // expect 192 to be last key in l.Keys()

	// make sure the cache has only half the capacity as we deleted half of them.
	cacheLen := l.Len()
	if capacity/2 != cacheLen {
		t.Fatalf("invalid len. expected %v, got %v", capacity/2, cacheLen)
	}

	// Keys - returns items from oldest to newest.
	for i, k := range l.Keys() {
		// last item should be `lastIndex` and make sure the other items are ordered
		if (i == cacheLen-1 && k != lastIndex) || (i < cacheLen-1 && k != i+lastIndex+1) {
			t.Fatalf("out of order key: %v %v %v", i, k, cacheLen-1)
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}

	// try to get the random item
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

func GetOldestRemoveOldestTest(t *testing.T, l simplelru.LRUCache, capacity int) {
	// add twice as much the capacity
	for i := 0; i < 2*capacity; i++ {
		l.Add(i, i)
	}

	k, _, ok := l.GetOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != capacity {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != capacity {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k.(int) != capacity+1 {
		t.Fatalf("bad: %v", k)
	}
}

func AddTest(t *testing.T, l simplelru.LRUCache, capacity int, evictCounter *int) {
	for i := 0; i < capacity; i++ {
		if l.Add(i, i) == true || *evictCounter != 0 {
			t.Errorf("should not have an eviction")
		}
	}
	if l.Add(capacity, capacity) == false || *evictCounter != 1 {
		t.Errorf("should have an eviction")
	}
}

func ContainsTest(t *testing.T, l simplelru.LRUCache, capacity int) {
	for i := 0; i < capacity; i++ {
		l.Add(i, i)
	}

	// contains should not update the recent-ness so this item will remain the oldest
	if !l.Contains(0) {
		t.Errorf("0 should be contained")
	}

	// oldest (0) should have been evicted
	l.Add(capacity, capacity)
	if l.Contains(0) {
		t.Errorf("Contains should not have updated recent-ness of 0")
	}
}

func PeekTest(t *testing.T, l simplelru.LRUCache, capacity int) {
	for i := 0; i < capacity; i++ {
		l.Add(i, i)
	}

	if v, ok := l.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(capacity, capacity)
	if l.Contains(0) {
		t.Errorf("should have been removed to make room for the new item")
	}
}
