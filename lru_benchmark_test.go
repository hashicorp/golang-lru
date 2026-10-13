package lru

import (
	"strconv"
	"testing"
)

const benchmarkCapacity = 1024

// BenchmarkLRU_Add measures the cost of adding items to the cache
// when the cache is not yet full.
func BenchmarkLRU_Add(b *testing.B) {
	cache, _ := New[int, int](benchmarkCapacity)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Add(i, i)
	}
}

// BenchmarkLRU_Get_Hit measures the cost of a cache hit.
// This represents the most common and performance-critical path.
func BenchmarkLRU_Get_Hit(b *testing.B) {
	cache, _ := New[int, int](benchmarkCapacity)

	// Pre-fill the cache to ensure all Get operations are hits.
	for i := 0; i < benchmarkCapacity; i++ {
		cache.Add(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Get(i % benchmarkCapacity)
	}
}

// BenchmarkLRU_Add_Eviction measures the cost of adding items
// when the cache is full and evictions occur.
func BenchmarkLRU_Add_Eviction(b *testing.B) {
	cache, _ := New[int, int](benchmarkCapacity)

	// Fill the cache to capacity.
	for i := 0; i < benchmarkCapacity; i++ {
		cache.Add(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Add(i+benchmarkCapacity, i)
	}
}

// BenchmarkLRU_Get_Parallel measures concurrent cache reads,
// reflecting real-world usage in multi-goroutine environments.
func BenchmarkLRU_Get_Parallel(b *testing.B) {
	cache, _ := New[string, string](benchmarkCapacity)

	for i := 0; i < benchmarkCapacity; i++ {
		key := strconv.Itoa(i)
		cache.Add(key, key)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			cache.Get(strconv.Itoa(i % benchmarkCapacity))
			i++
		}
	})
}
