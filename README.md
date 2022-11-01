golang-lru
==========

This provides the `lru` package which implements a fixed-size
thread safe LRU cache. It is based on the cache in Groupcache.

Documentation
=============

Full docs are available on [Godoc](https://pkg.go.dev/github.com/hashicorp/golang-lru)

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

Or use expiring caches as following:

```go
const EntryLifeTime = time.Minute
cache, _ := NewExpiringLRU(128, EntryLifeTime)
for i := 1; i < 256; i++ {
    cache.Add(i, nil)
}
// and run a background goroutine to clean up expired entries aggressively
go func() {
	LOOP:
		for {
			select {
			case <-shutdown:
				break LOOP
			case <-time.Tick(EntryLifeTime):
				cache.RemoveAllExpired()
			}
		}
}()
```
