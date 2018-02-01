package lru

import (
	"testing"
	"time"
)

// test that Add returns true/false if an eviction occurred
func TestLRUTTLAddNoTTL(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter += 1
	}

	l, err := NewTTLWithEvict(1, onEvicted)
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

// test that Add returns true/false if an eviction occurred
func TestLRUTTLAddWithTTL(t *testing.T) {
	evictCounter := 0
	onEvicted := func(k interface{}, v interface{}) {
		evictCounter += 1
		if v.(int) != evictCounter {
			t.Errorf("Eviction happened out of order. Got %v, expected %v", v.(int), evictCounter)
		}
	}

	l, err := NewTTLWithEvict(2, onEvicted)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if l.AddWithTTL(1, 1, time.Millisecond*5) == true {
		t.Errorf("should not have an eviction")
	}
	if l.AddWithTTL(2, 2, time.Millisecond*10) == true {
		t.Errorf("should have an eviction")
	}

	// Wait for TTLs to expire
	time.Sleep(25 * time.Millisecond)

	if evictCounter != 2 {
		t.Errorf("should have been 2 evictions")
	}
}
