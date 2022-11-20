package simplelru

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"
)

func BenchmarkExpirableLRU_Rand_NoExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, 0, 0)

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

func BenchmarkExpirableLRU_Freq_NoExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, 0, 0)

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

func BenchmarkExpirableLRU_Rand_WithExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, time.Millisecond*10, time.Millisecond*50)

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

func BenchmarkExpirableLRU_Freq_WithExpire(b *testing.B) {
	l := NewExpirableLRU[int64, int64](8192, nil, time.Millisecond*10, time.Millisecond*50)

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

func TestExpirableLRUInterface(t *testing.T) {
	var _ LRUCache[int, int] = &ExpirableLRU[int, int]{}
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

	if !reflect.DeepEqual(lc.Keys(), []string{"key1"}) {
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

func TestExpirableMultipleClose(t *testing.T) {
	lc := NewExpirableLRU[string, string](10, nil, 0, 0)
	lc.Close()
	// should not panic
	lc.Close()
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
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
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
	if !reflect.DeepEqual(evicted, []string{"key1", "val1"}) {
		t.Fatalf("value differs from expected")
	}

	// Purge, cache should be clean
	lc.Purge()
	if lc.Len() != 0 {
		t.Fatalf("length differs from expected")
	}
	if !reflect.DeepEqual(evicted, []string{"key1", "val1", "key2", "val2"}) {
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

	if !reflect.DeepEqual(lc.Keys(), []string{"key1"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	lc.Add("key2", "val2")
	if !reflect.DeepEqual(lc.Keys(), []string{"key1", "key2"}) {
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

	if !reflect.DeepEqual(lc.Keys(), []string{"key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
}

func ExampleExpirableLRU() {
	// make cache with 5ms TTL and 3 max keys, purge every 10ms
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

func getRand(tb testing.TB) int64 {
	out, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		tb.Fatal(err)
	}
	return out.Int64()
}
