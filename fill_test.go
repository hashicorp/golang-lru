package lru

import (
	"errors"
	"testing"
	"time"
)

type filler struct {
	value   int
	err     error
	blocker chan struct{}
	timeout time.Duration
}

func (fr *filler) Fill(key interface{}) (interface{}, time.Time, error) {
	fr.value++
	if fr.blocker != nil {
		<-fr.blocker
	}
	return fr.value, time.Now().Add(fr.timeout), fr.err
}

func TestFillingLRU(t *testing.T) {
	fr := filler{timeout: 10 * time.Second}
	fc := NewFillingCache(1, &fr)

	//should fill
	val, err := fc.Get("asdf")
	if val != 1 {
		t.Errorf("expected 1 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	//should returned cached version
	val, err = fc.Get("asdf")
	if val != 1 {
		t.Errorf("expected cached 1 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	//should fill
	val, err = fc.Get("something else")
	if val != 2 {
		t.Errorf("expected 2 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	//we expect this goes to 3, since the original asdf should now be evicted
	val, err = fc.Get("asdf")
	if val != 3 {
		t.Errorf("expected 3 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	// evict "asdf"
	val, err = fc.Get("something else")
	if val != 4 {
		t.Errorf("expected 4 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	// expire right away
	fr.timeout = 0
	val, err = fc.Get("asdf")
	if val != 5 {
		t.Errorf("expected 5 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	val, err = fc.Get("asdf")
	if val != 6 {
		t.Errorf("expected 6 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	// expire with a bit delay
	fr.timeout = 10 * time.Millisecond
	// evict "asdf"
	val, err = fc.Get("something else")
	if val != 7 {
		t.Errorf("expected 7 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}
	val, err = fc.Get("asdf")
	if val != 8 {
		t.Errorf("expected 8 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}
	time.Sleep(50 * time.Millisecond)
	val, err = fc.Get("asdf")
	if val != 9 {
		t.Errorf("expected 9 got %d", val)
	}
	if err != nil {
		t.Error("expected no errors")
	}

	// fill error
	fr.err = errors.New("fill error")
	fr.timeout = time.Second
	_, err = fc.Get("something else")
	if err == nil {
		t.Error("expect err %v", fr.err)
	}
}

func TestFillingLRUWaitForFilling(t *testing.T) {
	blocker := make(chan struct{}, 1)
	fr := filler{
		blocker: blocker,
		timeout: 10 * time.Second,
	}
	fc := NewFillingCache(1, &fr)

	chs := make([]chan struct{}, 100)
	for i := 0; i < len(chs); i++ {
		chs[i] = make(chan struct{}, 1)
		go func(ch chan struct{}) {
			val, err := fc.Get("asdf")
			if val != 1 {
				t.Errorf("expected 1 got %d", val)
			}
			if err != nil {
				t.Error("expected no errors")
			}
			ch <- struct{}{}
		}(chs[i])
	}

	for i := 0; i < len(chs); i++ {
		select {
		case <-chs[i]:
			t.Errorf("expected client %d to block", i)
		default:
		}
	}

	blocker <- struct{}{}

	tm := time.After(100 * time.Millisecond)
	for i := 0; i < len(chs); i++ {
		select {
		case <-chs[i]:
		case <-tm:
			t.Errorf("unexpected timeout")
		}
	}
}
