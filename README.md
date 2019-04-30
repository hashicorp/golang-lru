[![Build Status](https://travis-ci.org/hashicorp/golang-lru.svg?branch=master)](https://travis-ci.org/hashicorp/golang-lru)
[![Coverage Status](https://coveralls.io/repos/github/hashicorp/golang-lru/badge.svg?branch=master)](https://coveralls.io/github/hashicorp/golang-lru?branch=master)

golang-lru
==========

This provides the `lru` package which implements a fixed-size
thread safe LRU cache. It is based on the cache in Groupcache.

Documentation
=============

Full docs are available on [Godoc](http://godoc.org/github.com/hashicorp/golang-lru)

Example
=======

Using the LRU is very simple:

```go
l, _ := New(128)
for i := 0; i < 256; i++ {
    l.Add(i, nil)
}
if l.Len() != 128 {
    panic(fmt.Sprintf("bad len: %v", l.Len()))
}
```
