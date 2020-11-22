package lru

// RWLocker define base interface of sync.RWMutex
type RWLocker interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

// NoOpRWLocker is a dummy noop implementation of RWLocker interface
type NoOpRWLocker struct{}

// Lock perform noop Lock() operation
func (nop NoOpRWLocker) Lock() {}

// Unlock perform noop Unlock() operation
func (nop NoOpRWLocker) Unlock() {}

// RLock perform noop RLock() operation
func (nop NoOpRWLocker) RLock() {}

// RUnlock perform noop RUnlock() operation
func (nop NoOpRWLocker) RUnlock() {}
