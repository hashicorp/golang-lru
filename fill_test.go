package lru

import (
	"sync"
	"testing"
	"time"
)

func TestFillingLRU(t *testing.T) {

	l, err := NewFilling(1)
	i := 0

	l.fill = func(key interface{}) (interface{}, time.Time, error) {
		i++
		return i, time.Now().Add(time.Second), nil
	}

	//should fill
	val, err := l.Get("asdf")
	if val != 1 {
		t.Error("expected 1")
	}
	if err != nil {
		t.Error("expected no errors")
	}

	//should returned cached version
	val, err = l.Get("asdf")
	if val != 1 {
		t.Error("expected 1")
	}
	if err != nil {
		t.Error("expected no errors")
	}

	//should fill
	val, err = l.Get("something else")
	if val != 2 {
		t.Error("expected 2")
	}
	if err != nil {
		t.Error("expected no errors")
	}

	//we expect this goes to 3, since the original asdf should now be evicted
	val, err = l.Get("asdf")
	if val != 3 {
		t.Error("expected 3")
	}
	if err != nil {
		t.Error("expected no errors")
	}
}

//Test thundering horde
func TestFillingLRUThunderingHorde(t *testing.T) {

	l, _ := NewFilling(1)
	i := 0

	l.fill = func(key interface{}) (interface{}, time.Time, error) {
		i++
		return i, time.Now().Add(time.Second), nil
	}

	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			val, err := l.Get("asdf")
			if val != 1 {
				t.Error("expected 1")
			}
			if err != nil {
				t.Error("expected no errors")
			}

		}()
	}

	wg.Wait()
}

func TestFillingLRUThunderingHordeBadExpiration(t *testing.T) {

	l, _ := NewFilling(1)
	i := 0

	l.fill = func(key interface{}) (interface{}, time.Time, error) {
		i++
		return i, time.Now().Add(-time.Second), nil
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			wg.Done()
			_, _ = l.Get("asdf")

		}()
	}

	wg.Wait()
	//when they all trample and get a bad expiration, they continue to trample
	if i != 100 {
		t.Error("Expected 100")
	}
}
