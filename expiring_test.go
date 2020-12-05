package lru

import (
	"math/rand"
	"sort"
	"testing"
	"time"
)

func BenchmarkExpiring2Q_Rand(b *testing.B) {
	l, err := NewExpiring2Q(8192, 5*time.Minute)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.Int63() % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			_, ok := l.Get(trace[i])
			if ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkExpiring2Q_Freq(b *testing.B) {
	l, err := NewExpiring2Q(8192, 5*time.Minute)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = rand.Int63() % 16384
		} else {
			trace[i] = rand.Int63() % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		_, ok := l.Get(trace[i])
		if ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkExpiringARC_Rand(b *testing.B) {
	l, err := NewExpiringARC(8192, 5*time.Minute)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.Int63() % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			_, ok := l.Get(trace[i])
			if ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkExpiringARC_Freq(b *testing.B) {
	l, err := NewExpiringARC(8192, 5*time.Minute)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = rand.Int63() % 16384
		} else {
			trace[i] = rand.Int63() % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		_, ok := l.Get(trace[i])
		if ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkExpiringLRU_Rand(b *testing.B) {
	l, err := NewExpiringLRU(8192, 5*time.Minute)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		trace[i] = rand.Int63() % 32768
	}

	b.ResetTimer()

	var hit, miss int
	for i := 0; i < 2*b.N; i++ {
		if i%2 == 0 {
			l.Add(trace[i], trace[i])
		} else {
			_, ok := l.Get(trace[i])
			if ok {
				hit++
			} else {
				miss++
			}
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func BenchmarkExpiringLRU_Freq(b *testing.B) {
	l, err := NewExpiringLRU(8192, 5*time.Minute)
	if err != nil {
		b.Fatalf("err: %v", err)
	}

	trace := make([]int64, b.N*2)
	for i := 0; i < b.N*2; i++ {
		if i%2 == 0 {
			trace[i] = rand.Int63() % 16384
		} else {
			trace[i] = rand.Int63() % 32768
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.Add(trace[i], trace[i])
	}
	var hit, miss int
	for i := 0; i < b.N; i++ {
		_, ok := l.Get(trace[i])
		if ok {
			hit++
		} else {
			miss++
		}
	}
	b.Logf("hit: %d miss: %d ratio: %f", hit, miss, float64(hit)/float64(miss))
}

func TestExpiring2Q_RandomOps(t *testing.T) {
	size := 128
	l, err := NewExpiring2Q(size, 5*time.Minute)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n := 200000
	for i := 0; i < n; i++ {
		key := rand.Int63() % 512
		r := rand.Int63()
		switch r % 3 {
		case 0:
			l.Add(key, key)
		case 1:
			l.Get(key)
		case 2:
			l.Remove(key)
		}

		if l.Len() > size {
			t.Fatalf("bad ExpiringCache size: %d, expected: %d",
				l.Len(), size)
		}
	}
}

func TestExpiringARC_RandomOps(t *testing.T) {
	size := 128
	l, err := NewExpiringARC(size, 5*time.Minute)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n := 200000
	for i := 0; i < n; i++ {
		key := rand.Int63() % 512
		r := rand.Int63()
		switch r % 3 {
		case 0:
			l.Add(key, key)
		case 1:
			l.Get(key)
		case 2:
			l.Remove(key)
		}

		if l.Len() > size {
			t.Fatalf("bad ExpiringCache size: %d, expected: %d",
				l.Len(), size)
		}
	}
}

func TestExpiringLRU_RandomOps(t *testing.T) {
	size := 128
	l, err := NewExpiringLRU(size, 5*time.Minute)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n := 200000
	for i := 0; i < n; i++ {
		key := rand.Int63() % 512
		r := rand.Int63()
		switch r % 3 {
		case 0:
			l.Add(key, key)
		case 1:
			l.Get(key)
		case 2:
			l.Remove(key)
		}

		if l.Len() > size {
			t.Fatalf("bad ExpiringCache size: %d, expected: %d",
				l.Len(), size)
		}
	}
}

// Test eviction by least-recently-used (2-queue LRU suuport retaining frequently-used)
func TestExpiring2Q_EvictionByLRU(t *testing.T) {
	var ek, ev interface{}
	elru, err := NewExpiring2Q(3, 30*time.Second, EvictedCallback(func(k, v interface{}) {
		ek = k
		ev = v
	}))
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to freq-used list
	// 2 will remain in recent-used list
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// next add 3,4; verify 2, 3 will be evicted
	for i := 3; i < 5; i++ {
		evicted := elru.Add(i, i)
		k, v := ek.(int), ev.(int)
		if !evicted || k != (i-1) || v != (i-1) {
			t.Fatalf("(%v %v) should be evicted, but got (%v,%v)", i-1, i-1, k, v)
		}
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	// since 0, 1 are touched twice (write & read) so
	// they are in frequently used list, they are kept
	// and 2,3,4 only touched once (write), so they
	// moved thru "recent" list, with 2,3 evicted
	for i, v := range []int{0, 1, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {0,1,4} left, but found %v", elru.Keys())
		}
	}
}

// testTimer used to simulate time-elapse for expiration tests
type testTimer struct {
	t time.Time
}

func newTestTimer() *testTimer                { return &testTimer{time.Now()} }
func (tt *testTimer) Now() time.Time          { return tt.t }
func (tt *testTimer) Advance(d time.Duration) { tt.t = tt.t.Add(d) }

// Test eviction by ExpireAfterWrite
func TestExpiring2Q_ExpireAfterWrite(t *testing.T) {
	var ek, ev interface{}
	// use test timer for expiration
	tt := newTestTimer()
	elru, err := NewExpiring2Q(3, 30*time.Second, TimeTicker(tt.Now), EvictedCallback(
		func(k, v interface{}) {
			ek = k
			ev = v
		},
	))
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	// test timer ticks 20 seconds
	tt.Advance(20 * time.Second)
	// add fresher entry <2,2> to cache
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to freq-used list
	// 2 will remain in recent-used list
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// test timer advance another 15 seconds, entries <0,0>,<1,1> timeout & expire now,
	// so they should be evicted, although they are more recently retrieved than <2,2>
	tt.Advance(15 * time.Second)
	// next add 3,4; verify 0,1 will be evicted
	for i := 3; i < 5; i++ {
		evicted := elru.Add(i, i)
		k, v := ek.(int), ev.(int)
		if !evicted || k != (i-3) || v != (i-3) {
			t.Fatalf("(%v %v) should be evicted, but got (%v,%v)", i-3, i-3, k, v)
		}
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].(int) < keys[j].(int) })
	// althoug 0, 1 are touched twice (write & read) so
	// they are in frequently used list, they are evicted because expiration
	// and 2,3,4 will be kept
	for i, v := range []int{2, 3, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {2,3,4} left, but found %v", elru.Keys())
		}
	}
}

// Test eviction by ExpireAfterAccess: basically same access sequence as above case
// but different result because of ExpireAfterAccess
func TestExpiring2Q_ExpireAfterAccess(t *testing.T) {
	// use test timer for expiration
	tt := newTestTimer()
	elru, err := NewExpiring2Q(3, 30*time.Second, TimeTicker(tt.Now), ExpireAfterAccess)
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	// test timer ticks 20 seconds
	tt.Advance(20 * time.Second)
	// add fresher entry <2,2> to cache
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to freq-used list
	// also moved them to back in expire list with newer timestamp
	// 2 will remain in recent-used list
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// test timer advance another 15 seconds, none expired
	// and 2 in recent list
	tt.Advance(15 * time.Second)
	// next add 3,4; verify 2,3 will be evicted, because 0,1 in freq list, not expired
	for i := 3; i < 5; i++ {
		elru.Add(i, i)
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].(int) < keys[j].(int) })
	// and 0,1,4 will be kept
	for i, v := range []int{0, 1, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {0,1,4} left, but found %v", elru.Keys())
		}
	}
}

// Test eviction by ExpireAfterWrite
func TestExpiringARC_ExpireAfterWrite(t *testing.T) {
	var ek, ev interface{}
	// use test timer for expiration
	tt := newTestTimer()
	elru, err := NewExpiringARC(3, 30*time.Second, TimeTicker(tt.Now), EvictedCallback(
		func(k, v interface{}) {
			ek, ev = k, v
		},
	))
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	// test timer ticks 20 seconds
	tt.Advance(20 * time.Second)
	// add fresher entry <2,2> to cache
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to freq-used list
	// 2 will remain in recent-used list
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// test timer advance another 15 seconds, entries <0,0>,<1,1> timeout & expire now,
	// so they should be evicted, although they are more recently retrieved than <2,2>
	tt.Advance(15 * time.Second)
	// next add 3,4; verify 0,1 will be evicted
	for i := 3; i < 5; i++ {
		evicted := elru.Add(i, i)
		k, v := ek.(int), ev.(int)
		if !evicted || k != (i-3) || v != (i-3) {
			t.Fatalf("(%v %v) should be evicted, but got (%v,%v)", i-3, i-3, k, v)
		}
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].(int) < keys[j].(int) })
	// althoug 0, 1 are touched twice (write & read) so
	// they are in frequently used list, they are evicted because expiration
	// and 2,3,4 will be kept
	for i, v := range []int{2, 3, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {2,3,4} left, but found %v", elru.Keys())
		}
	}
}

// Test eviction by ExpireAfterAccess: basically same access sequence as above case
// but different result because of ExpireAfterAccess
func TestExpiringARC_ExpireAfterAccess(t *testing.T) {
	// use test timer for expiration
	tt := newTestTimer()
	elru, err := NewExpiringARC(3, 30*time.Second, TimeTicker(tt.Now), ExpireAfterAccess)
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	// test timer ticks 20 seconds
	tt.Advance(20 * time.Second)
	// add fresher entry <2,2> to cache
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to freq-used list
	// also moved them to back in expire list with newer timestamp
	// 2 will remain in recent-used list
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// test timer advance another 15 seconds, none expired
	// and 2 in recent list
	tt.Advance(15 * time.Second)
	// next add 3,4; verify 2,3 will be evicted, because 0,1 in freq list, not expired
	for i := 3; i < 5; i++ {
		elru.Add(i, i)
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].(int) < keys[j].(int) })
	// and 0,1,4 will be kept
	for i, v := range []int{0, 1, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {0,1,4} left, but found %v", elru.Keys())
		}
	}
}

// Test eviction by ExpireAfterWrite
func TestExpiringLRU_ExpireAfterWrite(t *testing.T) {
	var ek, ev interface{}
	// use test timer for expiration
	tt := newTestTimer()
	elru, err := NewExpiringLRU(3, 30*time.Second, TimeTicker(tt.Now), EvictedCallback(
		func(k, v interface{}) {
			ek, ev = k, v
		},
	))
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	// test timer ticks 20 seconds
	tt.Advance(20 * time.Second)
	// add fresher entry <2,2> to cache
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to freq-used list
	// 2 will remain in recent-used list
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// test timer advance another 15 seconds, entries <0,0>,<1,1> timeout & expire now,
	// so they should be evicted, although they are more recently retrieved than <2,2>
	tt.Advance(15 * time.Second)
	// next add 3,4; verify 0,1 will be evicted
	for i := 3; i < 5; i++ {
		evicted := elru.Add(i, i)
		k, v := ek.(int), ev.(int)
		if !evicted || k != (i-3) || v != (i-3) {
			t.Fatalf("(%v %v) should be evicted, but got (%v,%v)", i-3, i-3, k, v)
		}
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].(int) < keys[j].(int) })
	// althoug 0, 1 are touched twice (write & read) so
	// they are in frequently used list, they are evicted because expiration
	// and 2,3,4 will be kept
	for i, v := range []int{2, 3, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {2,3,4} left, but found %v", elru.Keys())
		}
	}
}

// Test eviction by ExpireAfterAccess: basically same access sequence as above case
// but different result because of ExpireAfterAccess
func TestExpiringLRU_ExpireAfterAccess(t *testing.T) {
	// use test timer for expiration
	tt := newTestTimer()
	elru, err := NewExpiringLRU(3, 30*time.Second, TimeTicker(tt.Now), ExpireAfterAccess)
	if err != nil {
		t.Fatalf("failed to create expiring LRU")
	}
	for i := 0; i < 2; i++ {
		elru.Add(i, i)
	}
	// test timer ticks 20 seconds
	tt.Advance(20 * time.Second)
	// add fresher entry <2,2> to cache
	elru.Add(2, 2)
	// Get(0),Get(1) will move 0, 1 to back of access list
	// also moved them to back in expire list with newer timestamp
	// access list will be 2,0,1
	for i := 0; i < 2; i++ {
		elru.Get(i)
	}
	// test timer advance another 15 seconds, none expired
	tt.Advance(15 * time.Second)
	// next add 3,4; verify 2,0 will be evicted
	for i := 3; i < 5; i++ {
		elru.Add(i, i)
	}
	if elru.Len() != 3 {
		t.Fatalf("Expiring LRU eviction failed, expected 3 entries left, but found %v", elru.Len())
	}
	keys := elru.Keys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].(int) < keys[j].(int) })
	// and 1,3,4 will be kept
	for i, v := range []int{1, 3, 4} {
		if v != keys[i] {
			t.Fatalf("Expiring LRU eviction failed, expected keys {1,3,4} left, but found %v", elru.Keys())
		}
	}
}

func TestExpiring2Q(t *testing.T) {
	l, err := NewExpiring2Q(128, 5*time.Minute)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for i := 0; i < 256; i++ {
		l.Add(i, i)
	}
	if l.Len() != 128 {
		t.Fatalf("bad len: %v", l.Len())
	}

	for i, k := range l.Keys() {
		if v, ok := l.Get(k); !ok || v != k || v != i+128 {
			t.Fatalf("bad key: %v", k)
		}
	}
	for i := 0; i < 128; i++ {
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be evicted")
		}
	}
	for i := 128; i < 256; i++ {
		_, ok := l.Get(i)
		if !ok {
			t.Fatalf("should not be evicted")
		}
	}
	for i := 128; i < 192; i++ {
		l.Remove(i)
		_, ok := l.Get(i)
		if ok {
			t.Fatalf("should be deleted")
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

// Test that Contains doesn't update recent-ness
func TestExpiring2Q_Contains(t *testing.T) {
	l, err := NewExpiring2Q(2, 5*time.Minute)
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
func TestExpiring2Q_Peek(t *testing.T) {
	l, err := NewExpiring2Q(2, 5*time.Minute)
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
