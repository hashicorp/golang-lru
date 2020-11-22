package lru

// Common interface of sync.RWMutex
type RWLocker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

// NoOpRWLocker is a dummy noop implementation of RWLocker interface
type NoOpRWLocker struct{}

func (nop NoOpRWLocker) Lock()    {}
func (nop NoOpRWLocker) Unlock()  {}
func (nop NoOpRWLocker) RLock()   {}
func (nop NoOpRWLocker) RUnlock() {}
