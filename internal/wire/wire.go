package wire

import (
	"errors"
	"sync"
)

var ErrClosed = errors.New("wire: closed")

// Wire broadcasts runtime messages to raw and merged Eino streams.
type Wire struct {
	mu     sync.Mutex
	closed bool
	merger merger
	raw    *broadcaster
	merged *broadcaster
}

func New() *Wire {
	return &Wire{
		raw:    newBroadcaster(),
		merged: newBroadcaster(),
	}
}

func (w *Wire) SubscribeRaw(capacity int) (*Subscription, error) {
	return w.raw.subscribe(capacity)
}

func (w *Wire) SubscribeMerged(capacity int) (*Subscription, error) {
	return w.merged.subscribe(capacity)
}

func (w *Wire) Send(message Message) error {
	if message == nil {
		return errors.New("wire: cannot send nil message")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrClosed
	}
	if err := w.raw.publish(message); err != nil {
		return err
	}

	ready, err := w.merger.push(message)
	if err != nil {
		return err
	}
	return w.publishMerged(ready)
}

func (w *Wire) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ErrClosed
	}
	return w.publishMerged(w.merger.flush())
}

func (w *Wire) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	if err := w.publishMerged(w.merger.flush()); err != nil {
		return err
	}
	w.closed = true
	w.raw.close()
	w.merged.close()
	return nil
}

func (w *Wire) publishMerged(messages []Message) error {
	for _, message := range messages {
		if err := w.merged.publish(message); err != nil {
			return err
		}
	}
	return nil
}
