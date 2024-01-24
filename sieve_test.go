package lru

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSieve(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter++
	}

	l, err := NewWithOpts[int, int](128, WithSieve[int, int](), WithCallback[int, int](onEvicted))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(i, i)
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if l.Cap() != 128 {
		t.Fatalf("expect %d, but %d", 128, l.Cap())
	}

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("bad key: %v", k)
		}
	}
	for i, v := range l.Values() {
		if v != i+128 {
			t.Fatalf("bad value: %v", v)
		}
	}
	for i := 0; i < 128; i++ {
		if _, ok := l.Get(i); ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		if _, ok := l.Get(i); !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		l.Remove(i)
		if _, ok := l.Get(i); ok {
			t.Fatalf("should be deleted")
		}
	}

	l.GetOldest()
	l.Get(192)       // expect 192 to be last key in l.Keys()
	l.RemoveOldest() // Remove oldest will set the 192 as visited, which would have not be removed.

	if _, ok := l.Get(192); !ok {
		t.Fatalf("should not have been evcited")
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}

	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

// test that Add returns true/false if an eviction occurred
func TestSieveAdd(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		evictCounter++
	}

	l, err := NewWithOpts[int, int](1, WithSieve[int, int](), WithCallback[int, int](onEvicted))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if l.Add(1, 1) == true || evictCounter != 0 {
		t.Errorf("should not have an eviction")
	}
	if l.Add(2, 2) == false || evictCounter != 1 {
		t.Errorf("should have an eviction")
	}
}

// test that Contains doesn't update recent-ness
func TestSieveContains(t *testing.T) {
	l, err := NewWithOpts[int, int](2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	l.Add(2, 2)
	if !l.Contains(1) {
		t.Errorf("1 should be contained")
	}

	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("Contains should not have updated recent-ness of 1")
	}
}

// test that ContainsOrAdd doesn't update recent-ness
func TestSieveContainsOrAdd(t *testing.T) {
	l, err := NewWithOpts[int, int](2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	l.Add(2, 2)
	contains, evict := l.ContainsOrAdd(1, 1)
	if !contains {
		t.Errorf("1 should be contained")
	}
	if evict {
		t.Errorf("nothing should be evicted here")
	}

	l.Add(3, 3)
	contains, evict = l.ContainsOrAdd(1, 1)
	if contains {
		t.Errorf("1 should not have been contained")
	}
	if !evict {
		t.Errorf("an eviction should have occurred")
	}
	if !l.Contains(1) {
		t.Errorf("now 1 should be contained")
	}
}

// test that PeekOrAdd doesn't update recent-ness
func TestSievePeekOrAdd(t *testing.T) {
	l, err := NewWithOpts[int, int](2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	l.Add(2, 2)
	previous, contains, evict := l.PeekOrAdd(1, 1)
	if !contains {
		t.Errorf("1 should be contained")
	}
	if evict {
		t.Errorf("nothing should be evicted here")
	}
	if previous != 1 {
		t.Errorf("previous is not equal to 1")
	}

	l.Add(3, 3)
	contains, evict = l.ContainsOrAdd(1, 1)
	if contains {
		t.Errorf("1 should not have been contained")
	}
	if !evict {
		t.Errorf("an eviction should have occurred")
	}
	if !l.Contains(1) {
		t.Errorf("now 1 should be contained")
	}
}

// test that Peek doesn't update recent-ness
func TestSievePeek(t *testing.T) {
	l, err := NewWithOpts[int, int](2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	l.Add(1, 1)
	l.Add(2, 2)
	if v, ok := l.Peek(1); !ok || v != 1 {
		t.Errorf("1 should be set to 1: %v, %v", v, ok)
	}

	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("should not have updated recent-ness of 1")
	}
}

// test that Resize can upsize and downsize
func TestSieveResize(t *testing.T) {
	onEvictCounter := 0
	onEvicted := func(k int, v int) {
		onEvictCounter++
	}

	l, err := NewWithOpts[int, int](2, WithSieve[int, int](), WithCallback[int, int](onEvicted))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Downsize
	l.Add(1, 1)
	l.Add(2, 2)
	evicted := l.Resize(1)
	if evicted != 1 {
		t.Errorf("1 element should have been evicted: %v", evicted)
	}
	if onEvictCounter != 1 {
		t.Errorf("onEvicted should have been called 1 time: %v", onEvictCounter)
	}

	l.Add(3, 3)
	if l.Contains(1) {
		t.Errorf("Element 1 should have been evicted")
	}

	// Upsize
	evicted = l.Resize(2)
	if evicted != 0 {
		t.Errorf("0 elements should have been evicted: %v", evicted)
	}

	l.Add(4, 4)
	if !l.Contains(3) || !l.Contains(4) {
		t.Errorf("Cache should have contained 2 elements")
	}
}

func (c *Cache[K, V]) checkKeys(t *testing.T, want []K) {
	t.Helper()
	got := c.Keys()

	existingKeys := make(map[K]struct{})
	for _, k := range got {
		existingKeys[k] = struct{}{}
	}

	if len(want) != len(existingKeys) {
		t.Errorf("Expected Size: %d Actual: %d", len(want), len(existingKeys))
	}

	for _, wnt := range want {
		if _, ok := existingKeys[wnt]; !ok {
			t.Errorf("Expected %s to be present. ", fmt.Sprint(wnt))
		}
	}
}

func TestSieve_EvictionSameKey(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		var evictedKeys []int

		cache, _ := NewWithOpts[int, struct{}](2, WithSieve[int, struct{}](), WithCallback[int, struct{}](func(key int, _ struct{}) {
			evictedKeys = append(evictedKeys, key)
		}))

		if evicted := cache.Add(1, struct{}{}); evicted {
			t.Error("First 1: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1})

		if evicted := cache.Add(2, struct{}{}); evicted {
			t.Error("2: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1, 2})

		if evicted := cache.Add(1, struct{}{}); evicted {
			t.Error("Second 1: got unexpected eviction")
		}
		cache.checkKeys(t, []int{2, 1})

		if evicted := cache.Add(3, struct{}{}); !evicted {
			t.Error("3: did not get expected eviction")
		}
		cache.checkKeys(t, []int{1, 3})

		want := []int{2}
		if !reflect.DeepEqual(evictedKeys, want) {
			t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
		}
	})

	t.Run("ContainsOrAdd", func(t *testing.T) {
		var evictedKeys []int

		cache, _ := NewWithOpts[int, struct{}](2, WithSieve[int, struct{}](), WithCallback[int, struct{}](func(key int, _ struct{}) {
			evictedKeys = append(evictedKeys, key)
		}))

		contained, evicted := cache.ContainsOrAdd(1, struct{}{})
		if contained {
			t.Error("First 1: got unexpected contained")
		}
		if evicted {
			t.Error("First 1: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1})

		contained, evicted = cache.ContainsOrAdd(2, struct{}{})
		if contained {
			t.Error("2: got unexpected contained")
		}
		if evicted {
			t.Error("2: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1, 2})

		contained, evicted = cache.ContainsOrAdd(1, struct{}{})
		if !contained {
			t.Error("Second 1: did not get expected contained")
		}
		if evicted {
			t.Error("Second 1: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1, 2})

		contained, evicted = cache.ContainsOrAdd(3, struct{}{})
		if contained {
			t.Error("3: got unexpected contained")
		}
		if !evicted {
			t.Error("3: did not get expected eviction")
		}
		cache.checkKeys(t, []int{2, 3})

		want := []int{1}
		if !reflect.DeepEqual(evictedKeys, want) {
			t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
		}
	})

	t.Run("PeekOrAdd", func(t *testing.T) {
		var evictedKeys []int

		cache, _ := NewWithOpts[int, struct{}](2, WithSieve[int, struct{}](), WithCallback[int, struct{}](func(key int, _ struct{}) {
			evictedKeys = append(evictedKeys, key)
		}))

		_, contained, evicted := cache.PeekOrAdd(1, struct{}{})
		if contained {
			t.Error("First 1: got unexpected contained")
		}
		if evicted {
			t.Error("First 1: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1})

		_, contained, evicted = cache.PeekOrAdd(2, struct{}{})
		if contained {
			t.Error("2: got unexpected contained")
		}
		if evicted {
			t.Error("2: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1, 2})

		_, contained, evicted = cache.PeekOrAdd(1, struct{}{})
		if !contained {
			t.Error("Second 1: did not get expected contained")
		}
		if evicted {
			t.Error("Second 1: got unexpected eviction")
		}
		cache.checkKeys(t, []int{1, 2})

		_, contained, evicted = cache.PeekOrAdd(3, struct{}{})
		if contained {
			t.Error("3: got unexpected contained")
		}
		if !evicted {
			t.Error("3: did not get expected eviction")
		}
		cache.checkKeys(t, []int{2, 3})

		want := []int{1}
		if !reflect.DeepEqual(evictedKeys, want) {
			t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
		}
	})

}
