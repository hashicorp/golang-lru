// Package lru provides three different LRU caches of varying sophistication.
//
// Cache is a simple LRU cache. It is based on the
// LRU implementation in groupcache:
// https://github.com/golang/groupcache/tree/master/lru
//
// TwoQueueCache tracks frequently used and recently used entries separately.
// This avoids a burst of accesses from taking out frequently used entries,
// at the cost of about 2x computational overhead and some extra bookkeeping.
//
// All caches in this package take locks while operating, and are therefore
// thread-safe for consumers.
package lru
