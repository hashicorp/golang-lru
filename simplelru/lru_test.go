package simplelru

import (
	"fmt"
	"sync"
	"testing"
	"time"
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

	if evictCounter != 128 {
		t.Fatalf("bad evict count: %v", evictCounter)
	}

	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("bad key: %v", k)
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
		t.Errorf("onEvict should have been called 1 time: %v", onEvictCounter)
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

func TestExpirableLRUNoPurge(t *testing.T) {
	lc := NewExpirableLRU[string, string](10, nil, 0, 0)

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok := lc.Peek("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	if !lc.Contains("key1") {
		t.Fatalf("should contain key1")
	}
	if lc.Contains("key2") {
		t.Fatalf("should not contain key2")
	}

	v, ok = lc.Peek("key2")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	if !equalSlices(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}

	if lc.Resize(0) != 0 {
		t.Fatalf("evicted count differs from expected")
	}
	if lc.Resize(2) != 0 {
		t.Fatalf("evicted count differs from expected")
	}
	lc.Add("key2", "val2")
	if lc.Resize(1) != 1 {
		t.Fatalf("evicted count differs from expected")
	}
}

func TestExpirableLRUWithPurge(t *testing.T) {
	var evicted []string
	lc := NewExpirableLRU(10, func(key string, value string) { evicted = append(evicted, key, value) }, 150*time.Millisecond, time.Millisecond*100)
	defer lc.Close()

	k, v, ok := lc.GetOldest()
	if k != "" {
		t.Fatalf("should be empty")
	}
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	lc.Add("key1", "val1")

	time.Sleep(100 * time.Millisecond) // not enough to expire
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Get("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	time.Sleep(200 * time.Millisecond) // expire
	v, ok = lc.Get("key1")
	if ok {
		t.Fatalf("should be false")
	}
	if v != "" {
		t.Fatalf("should be nil")
	}

	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}
	if !equalSlices(evicted, []string{"key1", "val1"}) {
		t.Fatalf("value differs from expected")
	}

	// add new entry
	lc.Add("key2", "val2")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	k, v, ok = lc.GetOldest()
	if k != "key2" {
		t.Fatalf("value differs from expected")
	}
	if v != "val2" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	// DeleteExpired, nothing deleted
	lc.deleteExpired()
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
	if !equalSlices(evicted, []string{"key1", "val1"}) {
		t.Fatalf("value differs from expected")
	}

	// Purge, cache should be clean
	lc.Purge()
	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}
	if !equalSlices(evicted, []string{"key1", "val1", "key2", "val2"}) {
		t.Fatalf("value differs from expected")
	}
}

func TestExpirableLRUWithPurgeEnforcedBySize(t *testing.T) {
	lc := NewExpirableLRU[string, string](10, nil, time.Hour, 0)
	defer lc.Close()

	for i := 0; i < 100; i++ {
		i := i
		lc.Add(fmt.Sprintf("key%d", i), fmt.Sprintf("val%d", i))
		v, ok := lc.Get(fmt.Sprintf("key%d", i))
		if v != fmt.Sprintf("val%d", i) {
			t.Fatalf("value differs from expected")
		}
		if !ok {
			t.Fatalf("should be true")
		}
		if lc.Len() > 20 {
			t.Fatalf("length should be less than 20")
		}
	}

	if lc.Len() != 10 {
		t.Fatalf("length differs from expected")
	}
}

func TestExpirableLRUConcurrency(t *testing.T) {
	lc := NewExpirableLRU[string, string](0, nil, 0, 0)
	wg := sync.WaitGroup{}
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func(i int) {
			lc.Add(fmt.Sprintf("key-%d", i/10), fmt.Sprintf("val-%d", i/10))
			wg.Done()
		}(i)
	}
	wg.Wait()
	if lc.Len() != 100 {
		t.Fatalf("length differs from expected")
	}
}

func TestExpirableLRUInvalidateAndEvict(t *testing.T) {
	var evicted int
	lc := NewExpirableLRU(-1, func(_, _ string) { evicted++ }, 0, 0)

	lc.Add("key1", "val1")
	lc.Add("key2", "val2")

	val, ok := lc.Get("key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if val != "val1" {
		t.Fatalf("value differs from expected")
	}
	if evicted != 0 {
		t.Fatalf("value differs from expected")
	}

	lc.Remove("key1")
	if evicted != 1 {
		t.Fatalf("value differs from expected")
	}
	val, ok = lc.Get("key1")
	if val != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestLoadingExpired(t *testing.T) {
	lc := NewExpirableLRU[string, string](0, nil, time.Millisecond*5, 0)

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok := lc.Peek("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	v, ok = lc.Get("key1")
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	time.Sleep(time.Millisecond * 10) // wait for entry to expire
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	} // but not purged

	v, ok = lc.Peek("key1")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	v, ok = lc.Get("key1")
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestExpirableLRURemoveOldest(t *testing.T) {
	lc := NewExpirableLRU[string, string](2, nil, 0, 0)

	k, v, ok := lc.RemoveOldest()
	if k != "" {
		t.Fatalf("should be empty")
	}
	if v != "" {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	ok = lc.Remove("non_existent")
	if ok {
		t.Fatalf("should be false")
	}

	lc.Add("key1", "val1")
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	v, ok = lc.Get("key1")
	if !ok {
		t.Fatalf("should be true")
	}
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}

	if !equalSlices(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	lc.Add("key2", "val2")
	if !equalSlices(lc.Keys(), []string{"key1", "key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 2 {
		t.Fatalf("length differs from expected")
	}

	k, v, ok = lc.RemoveOldest()
	if k != "key1" {
		t.Fatalf("value differs from expected")
	}
	if v != "val1" {
		t.Fatalf("value differs from expected")
	}
	if !ok {
		t.Fatalf("should be true")
	}

	if !equalSlices(lc.Keys(), []string{"key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
}

func Example_expirable_LRU() {
	// make cache with short TTL and 3 max keys, purgeEvery time.Millisecond * 10
	cache := NewExpirableLRU[string, string](3, nil, time.Millisecond*5, time.Millisecond*10)
	// expirable cache need to be closed after used
	defer cache.Close()

	// set value under key1.
	cache.Add("key1", "val1")

	// get value under key1
	r, ok := cache.Get("key1")

	// check for OK value
	if ok {
		fmt.Printf("value before expiration is found: %v, value: %q\n", ok, r)
	}

	// wait for cache to expire
	time.Sleep(time.Millisecond * 16)

	// get value under key1 after key expiration
	r, ok = cache.Get("key1")
	fmt.Printf("value after expiration is found: %v, value: %q\n", ok, r)

	// set value under key2, would evict old entry because it is already expired.
	cache.Add("key2", "val2")

	fmt.Printf("Cache len: %d\n", cache.Len())
	// Output:
	// value before expiration is found: true, value: "val1"
	// value after expiration is found: false, value: ""
	// Cache len: 1
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
