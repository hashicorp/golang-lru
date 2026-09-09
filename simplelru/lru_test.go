// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: MPL-2.0

package simplelru

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
	l, err := NewLRU(128, onEvicted)
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
		if ok := l.Remove(i); !ok {
			t.Fatalf("should be contained")
		}
		if ok := l.Remove(i); ok {
			t.Fatalf("should not be contained")
		}
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

func TestLRU_GetOldest_RemoveOldest(t *testing.T) {
	l, err := NewLRU[int, int](128, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	for i := 0; i < 256; i++ {
		l.Add(i, i)
	}
	k, _, ok := l.GetOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k != 128 {
		t.Fatalf("bad: %v", k)
	}

	k, _, ok = l.RemoveOldest()
	if !ok {
		t.Fatalf("missing")
	}
	if k != 129 {
		t.Fatalf("bad: %v", k)
	}
}

// Test that Add returns true/false if an eviction occurred
func TestLRU_Add(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		evictCounter++
	}

	l, err := NewLRU(1, onEvicted)
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

func TestLRU_AddIf(t *testing.T) {
	newer := func(old, new int) bool { return new > old }

	t.Run("inserts when missing", func(t *testing.T) {
		l, err := NewLRU[int, int](2, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		updated, evicted := l.AddIf(1, 10, newer)
		if !updated {
			t.Errorf("expected insert")
		}
		if evicted {
			t.Errorf("expected no eviction")
		}
		if v, ok := l.Peek(1); !ok || v != 10 {
			t.Errorf("expected 10, got %v, %v", v, ok)
		}
	})

	t.Run("replaces when replace returns true", func(t *testing.T) {
		l, err := NewLRU[int, int](2, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		l.Add(1, 1)
		updated, evicted := l.AddIf(1, 2, newer)
		if !updated {
			t.Errorf("expected replace")
		}
		if evicted {
			t.Errorf("expected no eviction")
		}
		if v, ok := l.Peek(1); !ok || v != 2 {
			t.Errorf("expected 2, got %v, %v", v, ok)
		}
	})

	t.Run("keeps old when replace returns false", func(t *testing.T) {
		l, err := NewLRU[int, int](2, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		l.Add(1, 5)
		updated, evicted := l.AddIf(1, 3, newer)
		if updated {
			t.Errorf("expected no update")
		}
		if evicted {
			t.Errorf("expected no eviction")
		}
		if v, ok := l.Peek(1); !ok || v != 5 {
			t.Errorf("expected 5, got %v, %v", v, ok)
		}
	})

	t.Run("nil replace keeps existing", func(t *testing.T) {
		l, err := NewLRU[int, int](2, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		l.Add(1, 1)
		updated, _ := l.AddIf(1, 2, nil)
		if updated {
			t.Errorf("expected no update with nil replace")
		}
		if v, ok := l.Peek(1); !ok || v != 1 {
			t.Errorf("expected 1, got %v, %v", v, ok)
		}

		updated, _ = l.AddIf(2, 2, nil)
		if !updated {
			t.Errorf("expected insert of missing key with nil replace")
		}
	})

	t.Run("rejected update does not change recency", func(t *testing.T) {
		l, err := NewLRU[int, int](2, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		l.Add(1, 10)
		l.Add(2, 2)
		l.AddIf(1, 1, newer)
		l.Add(3, 3)
		if l.Contains(1) {
			t.Errorf("rejected update should not have refreshed recency of 1")
		}
	})

	t.Run("accepted update refreshes recency", func(t *testing.T) {
		l, err := NewLRU[int, int](2, nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		l.Add(1, 1)
		l.Add(2, 2)
		l.AddIf(1, 10, newer)
		l.Add(3, 3)
		if !l.Contains(1) {
			t.Errorf("accepted update should have refreshed recency of 1")
		}
		if l.Contains(2) {
			t.Errorf("2 should have been evicted")
		}
		if v, ok := l.Peek(1); !ok || v != 10 {
			t.Errorf("expected 10, got %v, %v", v, ok)
		}
	})

	t.Run("insert can evict", func(t *testing.T) {
		evictCounter := 0
		l, err := NewLRU(1, func(k int, v int) { evictCounter++ })
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		l.Add(1, 1)
		updated, evicted := l.AddIf(2, 2, newer)
		if !updated || !evicted || evictCounter != 1 {
			t.Errorf("expected insert with eviction, updated=%v evicted=%v count=%d", updated, evicted, evictCounter)
		}
	})
}

// Test that Contains doesn't update recent-ness
func TestLRU_Contains(t *testing.T) {
	l, err := NewLRU[int, int](2, nil)
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

// Test that Peek doesn't update recent-ness
func TestLRU_Peek(t *testing.T) {
	l, err := NewLRU[int, int](2, nil)
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

// Test that Resize can upsize and downsize
func TestLRU_Resize(t *testing.T) {
	onEvictCounter := 0
	onEvicted := func(k int, v int) {
		onEvictCounter++
	}
	l, err := NewLRU(2, onEvicted)
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

func (c *LRU[K, V]) wantKeys(t *testing.T, want []K) {
	t.Helper()
	got := c.Keys()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong keys got: %v, want: %v ", got, want)
	}
}

func TestCache_EvictionSameKey(t *testing.T) {
	var evictedKeys []int

	cache, _ := NewLRU(
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
}
