// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lru

import (
	"testing"
)

func Benchmark_Rand(b *testing.B) {
	var fn = func(b *testing.B, l *Cache[int64, int64]) {
		trace := make([]int64, b.N*2)
		for i := 0; i < b.N*2; i++ {
			trace[i] = getRand(b) % 32768
		}

		b.ResetTimer()

		var hit, miss int
		for i := 0; i < 2*b.N; i++ {
			if i%2 == 0 {
				l.Add(trace[i], trace[i])
			} else {
				if _, ok := l.Get(trace[i]); ok {
					hit++
				} else {
					miss++
				}
			}
		}

		b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
	}

	b.Run("Benchmark with LRU ", func(b *testing.B) {
		l, err := New[int64, int64](8192)
		if err != nil {
			b.Fatalf("err: %v", err)
		}

		fn(b, l)
	})

	b.Run("Benchmark with Sieve ", func(b *testing.B) {
		l, err := NewWithOpts[int64, int64](8192, WithSieve[int64, int64]())
		if err != nil {
			b.Fatalf("err: %v", err)
		}

		fn(b, l)
	})
}

func BenchmarkLRU_Freq(b *testing.B) {
	var fn = func(b *testing.B, l *Cache[int64, int64]) {
		trace := make([]int64, b.N*2)
		for i := 0; i < b.N*2; i++ {
			if i%2 == 0 {
				trace[i] = getRand(b) % 16384
			} else {
				trace[i] = getRand(b) % 32768
			}
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			l.Add(trace[i], trace[i])
		}
		var hit, miss int
		for i := 0; i < b.N; i++ {
			if _, ok := l.Get(trace[i]); ok {
				hit++
			} else {
				miss++
			}
		}
		b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(hit+miss))
	}

	b.Run("Benchmark with LRU ", func(b *testing.B) {
		l, err := New[int64, int64](8192)
		if err != nil {
			b.Fatalf("err: %v", err)
		}

		fn(b, l)
	})

	b.Run("Benchmark with Sieve ", func(b *testing.B) {
		l, err := NewWithOpts[int64, int64](8192, WithSieve[int64, int64]())
		if err != nil {
			b.Fatalf("err: %v", err)
		}

		fn(b, l)
	})
}

// test that Add returns true/false if an eviction occurred
func TestAdd(t *testing.T) {
	var evictCounter = 0
	var add = func(t *testing.T, c *Cache[int, int]) {
		if c.Add(1, 1) == true || evictCounter != 0 {
			t.Errorf("should not have an eviction")
		}
		if c.Add(2, 2) == false || evictCounter != 1 {
			t.Errorf("should have an eviction")
		}
	}

	onEvicted := func(k int, v int) {
		evictCounter++
	}

	l, err := NewWithEvict(1, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	t.Run("LRU add", func(t1 *testing.T) {
		evictCounter = 0
		add(t1, l)
	})

	l, err = NewWithOpts[int, int](1, WithSieve[int, int](), WithCallback[int, int](onEvicted))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	t.Run("Sieve add", func(t1 *testing.T) {
		evictCounter = 0
		add(t1, l)
	})
}

// test that ContainsOrAdd doesn't update recent-ness
func TestContainsOrAdd(t *testing.T) {
	var containsOrAdd = func(t *testing.T, l *Cache[int, int]) {
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

	t.Run(" LRU ContainsOrAdd ", func(t *testing.T) {
		l, err := New[int, int](2)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		containsOrAdd(t, l)
	})

	t.Run(" Sieve ContainsOrAdd ", func(t *testing.T) {
		l, err := NewWithOpts[int, int](2, WithSieve[int, int]())
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		containsOrAdd(t, l)
	})
}

// test that PeekOrAdd doesn't update recent-ness
func TestPeekOrAdd(t *testing.T) {
	var peekOrAdd = func(t *testing.T, l *Cache[int, int]) {
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

	t.Run("LRU PeekOrAdd", func(t *testing.T) {
		l, err := New[int, int](2)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		peekOrAdd(t, l)
	})

	t.Run("Sieve PeekOrAdd", func(t *testing.T) {
		l, err := NewWithOpts[int, int](2)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		peekOrAdd(t, l)
	})
}

// test that Peek doesn't update recent-ness
func TestPeek(t *testing.T) {
	var peek = func(t *testing.T, l *Cache[int, int]) {
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

	t.Run("LRU Peek", func(t *testing.T) {
		l, err := New[int, int](2)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		peek(t, l)
	})

	t.Run("Sieve Peek", func(t *testing.T) {
		l, err := NewWithOpts[int, int](2)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		peek(t, l)
	})
}

// test that Resize can upsize and downsize
func TestResize(t *testing.T) {
	var onEvictCounter int
	var resize = func(t *testing.T, l *Cache[int, int]) {
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

	t.Run("LRU resize", func(t *testing.T) {
		onEvictCounter = 0
		onEvicted := func(k int, v int) {
			onEvictCounter++
		}
		l, err := NewWithEvict(2, onEvicted)
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		resize(t, l)
	})

	t.Run("Sieve resize", func(t *testing.T) {
		onEvictCounter = 0
		onEvicted := func(k int, v int) {
			onEvictCounter++
		}

		l, err := NewWithOpts[int, int](2, WithSieve[int, int](), WithCallback[int, int](onEvicted))
		if err != nil {
			t.Fatalf("err: %v", err)
		}

		resize(t, l)
	})
}
