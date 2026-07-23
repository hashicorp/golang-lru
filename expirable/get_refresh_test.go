package expirable

import (
	"testing"
	"time"
)

func TestGetRefreshesTTL(t *testing.T) {
	c := NewLRU[string, int](10, nil, 80*time.Millisecond)
	c.Add("a", 1)
	time.Sleep(50 * time.Millisecond)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("first get: %v %v", v, ok)
	}
	// Without refresh, another 50ms would expire (80ms total from add).
	// With refresh on Get, should still be alive after another 50ms.
	time.Sleep(50 * time.Millisecond)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("expected sliding TTL to keep entry alive, got %v %v", v, ok)
	}
}
