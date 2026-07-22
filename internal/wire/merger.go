package wire

import "fmt"

type merger struct {
	buffer Mergeable
}

// push 尝试把下一块消息合并进当前缓冲区，如果无法合并，则返回当前缓冲区的消息，并将下一块消息作为新的缓冲区
func (m *merger) push(message Message) ([]Message, error) {
	if message == nil {
		return nil, fmt.Errorf("wire: cannot merge nil message")
	}

	next, mergeable := message.(Mergeable)
	if !mergeable {
		ready := m.flush()
		return append(ready, message), nil
	}

	if m.buffer == nil {
		m.buffer = next.Clone()
		return nil, nil
	}

	if m.buffer.MergeInPlace(next) {
		return nil, nil
	}

	ready := []Message{m.buffer}
	m.buffer = next.Clone()
	return ready, nil
}

// flush 将当前缓冲区的消息返回，并清空缓冲区.不再等待后续 chunk，把当前缓冲作为完整消息输出。
func (m *merger) flush() []Message {
	if m.buffer == nil {
		return nil
	}

	message := m.buffer
	m.buffer = nil
	return []Message{message}
}
