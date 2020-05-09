package simplelru

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestExpirableLRUInterface(t *testing.T) {
	var _ LRUCache = &ExpirableLRU{}
}

func TestExpirableLRUNoPurge(t *testing.T) {
	lc := NewExpirableLRU(10, nil, 0, 0)

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
	if v != nil {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	if !reflect.DeepEqual(lc.Keys(), []interface{}{"key1"}) {
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
	lc := NewExpirableLRU(10, func(key interface{}, value interface{}) { evicted = append(evicted, key.(string), value.(string)) }, 150*time.Millisecond, time.Millisecond*100)
	defer lc.Close()

	k, v, ok := lc.GetOldest()
	if k != nil {
		t.Fatalf("should be empty")
	}
	if v != nil {
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
	if v != nil {
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
	lc.DeleteExpired()
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
	lc := NewExpirableLRU(10, nil, time.Hour, 0)
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
	lc := NewExpirableLRU(0, nil, 0, 0)
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
	lc := NewExpirableLRU(-1, func(_, _ interface{}) { evicted++ }, 0, 0)

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
	if val != nil {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestLoadingExpired(t *testing.T) {
	lc := NewExpirableLRU(0, nil, time.Millisecond*5, 0)

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
	if v != nil {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}

	v, ok = lc.Get("key1")
	if v != nil {
		t.Fatalf("should be empty")
	}
	if ok {
		t.Fatalf("should be false")
	}
}

func TestExpirableLRURemoveOldest(t *testing.T) {
	lc := NewExpirableLRU(2, nil, 0, 0)

	k, v, ok := lc.RemoveOldest()
	if k != nil {
		t.Fatalf("should be empty")
	}
	if v != nil {
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

	if !reflect.DeepEqual(lc.Keys(), []interface{}{"key1"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}

	lc.Add("key2", "val2")
	if !reflect.DeepEqual(lc.Keys(), []interface{}{"key1", "key2"}) {
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

	if !reflect.DeepEqual(lc.Keys(), []interface{}{"key2"}) {
		t.Fatalf("value differs from expected")
	}
	if lc.Len() != 1 {
		t.Fatalf("length differs from expected")
	}
}
