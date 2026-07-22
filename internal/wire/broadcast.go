package wire

import (
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// Subscription exposes Wire messages through Eino's read-once stream abstraction.
type Subscription struct {
	*schema.StreamReader[Message]
	once   sync.Once
	cancel func()
}

// Close releases both the Eino reader and the corresponding broadcast subscriber.
func (s *Subscription) Close() {
	s.once.Do(func() {
		s.StreamReader.Close()
		s.cancel()
	})
}

type broadcaster struct {
	mu     sync.Mutex
	nextID uint64
	closed bool
	sinks  map[uint64]*subscriberSink
}

func newBroadcaster() *broadcaster {
	return &broadcaster{sinks: make(map[uint64]*subscriberSink)}
}

func (b *broadcaster) subscribe(capacity int) (*Subscription, error) {
	if capacity < 0 {
		return nil, fmt.Errorf("wire: subscription capacity cannot be negative")
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, ErrClosed
	}

	id := b.nextID
	b.nextID++
	reader, writer := schema.Pipe[Message](capacity)
	sink := newSubscriberSink(writer)
	b.sinks[id] = sink

	return &Subscription{
		StreamReader: reader,
		cancel: func() {
			b.mu.Lock()
			defer b.mu.Unlock()
			if sink, ok := b.sinks[id]; ok {
				delete(b.sinks, id)
				sink.close()
			}
		},
	}, nil
}

func (b *broadcaster) publish(message Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return ErrClosed
	}

	for id, sink := range b.sinks {
		if !sink.enqueue(message) {
			delete(b.sinks, id)
		}
	}
	return nil
}

func (b *broadcaster) close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for id, sink := range b.sinks {
		sink.close()
		delete(b.sinks, id)
	}
}

// subscriberSink decouples a non-blocking Kimi broadcast from an Eino stream reader.
// Kimi's Python BroadcastQueue is unbounded, so slow UI consumers must not block the soul.
type subscriberSink struct {
	mu      sync.Mutex
	ready   *sync.Cond
	queue   []Message
	closing bool
	writer  *schema.StreamWriter[Message]
}

func newSubscriberSink(writer *schema.StreamWriter[Message]) *subscriberSink {
	sink := &subscriberSink{writer: writer}
	sink.ready = sync.NewCond(&sink.mu)
	go sink.pump()
	return sink
}

func (s *subscriberSink) enqueue(message Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return false
	}
	s.queue = append(s.queue, message)
	s.ready.Signal()
	return true
}

func (s *subscriberSink) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closing {
		return
	}
	s.closing = true
	s.ready.Signal()
}

func (s *subscriberSink) pump() {
	defer s.writer.Close()
	for {
		s.mu.Lock()
		for len(s.queue) == 0 && !s.closing {
			s.ready.Wait()
		}
		if len(s.queue) == 0 && s.closing {
			s.mu.Unlock()
			return
		}
		message := s.queue[0]
		s.queue[0] = nil
		s.queue = s.queue[1:]
		s.mu.Unlock()

		if s.writer.Send(message, nil) {
			s.mu.Lock()
			s.queue = nil
			s.closing = true
			s.mu.Unlock()
			return
		}
	}
}
