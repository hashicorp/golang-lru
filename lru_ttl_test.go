package lru

import (
	"testing"
	"time"

	"github.com/hashicorp/golang-lru/testutils"
)

func TestLRUWithTTL(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		if k != v {
			t.Fatalf("Evict values not equal (%v!=%v)", k, v)
		}
		evictCounter++
	}

	l, err := NewTTLWithEvict(128, 1*time.Millisecond, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// test the LRU basic functions
	testutils.BasicTest(t, l, 128, &evictCounter)

	l.Purge()

	// test TTL
	evictCounter = 0
	for i := 0; i < 128; i++ {
		l.Add(i, i)

		// make the items are evicted once TTL is met
		time.Sleep(5 * time.Millisecond)

		if l.Contains(i) {
			t.Fatalf("item should have been evicted as TTL is met: %v", i)
		}
	}

	// create a new cache with large capacity
	evictCounter = 0
	l, err = NewTTLWithEvict(20000, 1*time.Millisecond, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 20000; i++ {
		l.Add(i, i)

		// make the items are evicted once TTL is met
		time.Sleep(5 * time.Millisecond)

		if _, ok := l.Get(i); ok {
			t.Fatalf("item should have been evicted as TTL is met: %v", i)
		}
	}

	// all the items should have been evicted
	cacheLen := l.Len()
	if cacheLen > 0 {
		t.Fatalf("bad len, expected: 0, got: %v", cacheLen)
	}
}

func TestLRUWithTTLGetOldestRemoveOldest(t *testing.T) {
	l, err := NewTTL(256, 1*time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	testutils.GetOldestRemoveOldestTest(t, l, 256)
}

// Test that Add returns true/false if an eviction occurred
func TestLRUWithTTLAdd(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter++
	}

	l, err := NewTTLWithEvict(1, 1*time.Second, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	testutils.AddTest(t, l, 1, &evictCounter)
}

// Test that Contains doesn't update recent-ness
func TestLRUWithTTLContains(t *testing.T) {
	l, err := NewTTL(2, 1*time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	testutils.ContainsTest(t, l, 2)
}

// TestLRUWithTTLPeek that Peek doesn't update recent-ness
func TestLRUWithTTLPeek(t *testing.T) {
	l, err := NewTTL(2, 1*time.Second)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	testutils.PeekTest(t, l, 2)
}
