// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package simplelru

import (
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
	l, err := NewSieve(128, onEvicted)
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

	l.Purge()
	if l.Len() != 0 {
		t.Fatalf("bad len: %v", l.Len())
	}
	if _, ok := l.Get(200); ok {
		t.Fatalf("should contain nothing")
	}
}

func TestSieve_GetOldest_RemoveOldest(t *testing.T) {
	l, err := NewSieve[int, int](128, nil)
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
func TestSieve_Add(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k int, v int) {
		evictCounter++
	}

	l, err := NewSieve(1, onEvicted)
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

// Test that Contains doesn't update recent-ness
func TestSieve_Contains(t *testing.T) {
	l, err := NewSieve[int, int](2, nil)
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
func TestSieve_Peek(t *testing.T) {
	l, err := NewSieve[int, int](2, nil)
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
func TestSieve_Resize(t *testing.T) {
	onEvictCounter := 0
	onEvicted := func(k int, v int) {
		onEvictCounter++
	}
	l, err := NewSieve(2, onEvicted)
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

func TestSieve_EvictionSameKey(t *testing.T) {
	var evictedKeys []int

	cache, _ := NewSieve(
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

	for _, k := range []int{1, 2} {
		present, visited := cache.visited(k)
		if !present || visited {
			t.Errorf("Expecting both the keys to be present without visited being set present:%v visited:%v", present, visited)
		}
	}

	if evicted := cache.Add(1, struct{}{}); evicted {
		t.Error("Second 1: got unexpected eviction")
	}

	present, visited := cache.visited(1)
	if !present || !visited {
		t.Errorf("Expecting the key to be visited and present , Actual  present:%v visited:%v", present, visited)
	}

	present, visited = cache.visited(2)
	if !present || visited {
		t.Errorf("Expecting the key to be not visited and present , Actual  present:%v visited:%v", present, visited)
	}

	if evicted := cache.Add(2, struct{}{}); evicted {
		t.Error("Second 1: got unexpected eviction")
	}

	for _, k := range []int{2, 1} {
		present, visited := cache.visited(k)
		if !present || !visited {
			t.Errorf("Expecting both the keys to be present and visited, Actual present:%v visited:%v", present, visited)
		}
	}

	if evicted := cache.Add(3, struct{}{}); !evicted {
		t.Error("3: did not get expected eviction")
	}

	cache.wantKeys(t, []int{2, 3})
	if evicted := cache.Add(4, struct{}{}); !evicted {
		t.Error("4: did not get expected eviction")
	}

	cache.wantKeys(t, []int{3, 4})
	if _, ok := cache.Get(3); !ok {
		t.Errorf("3 should be present in cache")
	}

	present, visited = cache.visited(3)
	if !present || !visited {
		t.Errorf("Expecting the key to be present and visited, Actual present:%v visited:%v", present, visited)
	}

	present, visited = cache.visited(4)
	if !present || visited {
		t.Errorf("Expecting the key to be present and not visited, Actual present:%v visited:%v", present, visited)
	}

	if evicted := cache.Add(1, struct{}{}); !evicted {
		t.Error("1: did not get expected eviction")
	}

	cache.wantKeys(t, []int{3, 1})

	want := []int{1, 2, 4}
	if !reflect.DeepEqual(evictedKeys, want) {
		t.Errorf("evictedKeys got: %v want: %v", evictedKeys, want)
	}
}


