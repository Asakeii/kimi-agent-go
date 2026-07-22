package wire

import (
	"context"
	"sync"
)

// future is a single-assignment result used by request messages.
// Its zero value is ready for use so requests remain safe after JSON decoding.
type future[T any] struct {
	once     sync.Once
	mu       sync.RWMutex
	done     chan struct{}
	value    T
	err      error
	resolved bool
}

func (f *future[T]) initialize() {
	f.once.Do(func() { f.done = make(chan struct{}) })
}

func (f *future[T]) wait(ctx context.Context) (T, error) {
	f.initialize()
	select {
	case <-f.done:
		f.mu.RLock()
		defer f.mu.RUnlock()
		return f.value, f.err
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}
}

func (f *future[T]) resolve(value T, err error) bool {
	f.initialize()
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.resolved {
		return false
	}
	f.value = value
	f.err = err
	f.resolved = true
	close(f.done)
	return true
}

func (f *future[T]) isResolved() bool {
	f.initialize()
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.resolved
}

func (f *future[T]) resolvedValue() (T, bool) {
	f.initialize()
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.value, f.resolved
}
