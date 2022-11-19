golang-lru
==========

This provides the `lru` package which implements a fixed-size
thread safe LRU cache. It is based on the cache in Groupcache.

Documentation
=============

Full docs are available on [Go Packages](https://pkg.go.dev/github.com/hashicorp/golang-lru/v2)

LRU cache example
=================

```go
package main

import (
	"fmt"

	"github.com/hashicorp/golang-lru/v2"
)

func main() {
	l, _ := lru.New[int, *string](128)
	for i := 0; i < 256; i++ {
		l.Add(i, nil)
	}

	if l.Len() != 128 {
		panic(fmt.Sprintf("bad len: %v", l.Len()))
	}
}
```

Expirable LRU cache example
===========================

```go
package main

import (
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/simplelru"
)

func main() {
	// make cache with short TTL and 3 max keys, purgeEvery time.Millisecond * 10
	cache := simplelru.NewExpirableLRU[string, string](3, nil, time.Millisecond*5, time.Millisecond*10)
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
```
