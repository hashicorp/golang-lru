// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lru

import (
	"reflect"
	"testing"
)

func TestLRU(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter++
	}

	l, err := NewWithEvict(128, onEvicted)
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

	l.Get(192) // expect 192 to be last key in l.Keys()

	for i, k := range l.Keys() {
		if (i < 63 && k != i+193) || (i == 63 && k != 192) {
			t.Fatalf("out of order key: %v", k)
		}
	}

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

// test that Contains doesn't update recent-ness
func TestLRUContains(t *testing.T) {
	l, err := New[int, int](2)
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

func (c *Cache[K, V]) wantKeys(t *testing.T, want []K) {
	t.Helper()
	got := c.Keys()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong keys got: %v, want: %v ", got, want)
	}
}

func TestCache_EvictionSameKey(t *testing.T) {
	t.Run("Add", func(t *testing.T) {
		var evictedKeys []int

		cache, _ := NewWithEvict(
			2,
			func(key int, _ struct{}) {
				evictedKeys = append(evictedKeys, key)
			})

		if evicted := cache.Add(1, struct{}{}); evicted {
			t.Error("First 1: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1})

		if evicted := cache.Add(2, struct{}{}); evicted {
			t.Error("2: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1, 2})

		if evicted := cache.Add(1, struct{}{}); evicted {
			t.Error("Second 1: got unexpected eviction")
		}
		cache.wantKeys(t, []int{2, 1})

		if evicted := cache.Add(3, struct{}{}); !evicted {
			t.Error("3: did not get expected eviction")
		}
		cache.wantKeys(t, []int{1, 3})

		want := []int{2}
		if !reflect.DeepEqual(evictedKeys, want) {
			t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
		}
	})

	t.Run("ContainsOrAdd", func(t *testing.T) {
		var evictedKeys []int

		cache, _ := NewWithEvict(
			2,
			func(key int, _ struct{}) {
				evictedKeys = append(evictedKeys, key)
			})

		contained, evicted := cache.ContainsOrAdd(1, struct{}{})
		if contained {
			t.Error("First 1: got unexpected contained")
		}
		if evicted {
			t.Error("First 1: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1})

		contained, evicted = cache.ContainsOrAdd(2, struct{}{})
		if contained {
			t.Error("2: got unexpected contained")
		}
		if evicted {
			t.Error("2: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1, 2})

		contained, evicted = cache.ContainsOrAdd(1, struct{}{})
		if !contained {
			t.Error("Second 1: did not get expected contained")
		}
		if evicted {
			t.Error("Second 1: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1, 2})

		contained, evicted = cache.ContainsOrAdd(3, struct{}{})
		if contained {
			t.Error("3: got unexpected contained")
		}
		if !evicted {
			t.Error("3: did not get expected eviction")
		}
		cache.wantKeys(t, []int{2, 3})

		want := []int{1}
		if !reflect.DeepEqual(evictedKeys, want) {
			t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
		}
	})

	t.Run("PeekOrAdd", func(t *testing.T) {
		var evictedKeys []int

		cache, _ := NewWithEvict(
			2,
			func(key int, _ struct{}) {
				evictedKeys = append(evictedKeys, key)
			})

		_, contained, evicted := cache.PeekOrAdd(1, struct{}{})
		if contained {
			t.Error("First 1: got unexpected contained")
		}
		if evicted {
			t.Error("First 1: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1})

		_, contained, evicted = cache.PeekOrAdd(2, struct{}{})
		if contained {
			t.Error("2: got unexpected contained")
		}
		if evicted {
			t.Error("2: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1, 2})

		_, contained, evicted = cache.PeekOrAdd(1, struct{}{})
		if !contained {
			t.Error("Second 1: did not get expected contained")
		}
		if evicted {
			t.Error("Second 1: got unexpected eviction")
		}
		cache.wantKeys(t, []int{1, 2})

		_, contained, evicted = cache.PeekOrAdd(3, struct{}{})
		if contained {
			t.Error("3: got unexpected contained")
		}
		if !evicted {
			t.Error("3: did not get expected eviction")
		}
		cache.wantKeys(t, []int{2, 3})

		want := []int{1}
		if !reflect.DeepEqual(evictedKeys, want) {
			t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
		}
	})
}
