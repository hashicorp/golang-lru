golang-lru
==========

This provides the `lru` package which implements a fixed-size
thread safe LRU cache. It is based on the cache in Groupcache.

Documentation
=============

Full docs are available on [Go Packages](https://pkg.go.dev/github.com/hashicorp/golang-lru/v2)

Example
=======

Using the LRU is very simple:

```go
package main

import (
	"fmt"
	"github.com/hashicorp/golang-lru/v2"
)

func main() {
	l, _ := lru.New[int, any](128)
	for i := 0; i < 256; i++ {
		l.Add(i, nil)
	}

	fmt.Printf("the lru length is %d\n", l.Len())

	ok := l.Contains(127)
	if !ok {
		fmt.Printf("the key is not found\n")
	}

	val, ok := l.Get(129)
	if ok {
		fmt.Printf("the value is %v\n", val)
	}

	l.Resize(64)
	fmt.Printf("the lru length is %d\n", l.Len())

	// Output:
	// the lru length is 128
	// the key is not found
	// the value is <nil>
	// the lru length is 64
}
```
