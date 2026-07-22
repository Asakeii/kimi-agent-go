package wire

import "testing"

func TestMergerCombinesTextWithoutChangingRawMessage(t *testing.T) {
	m := &merger{}
	first := NewTextPart("Hello")

	ready, err := m.push(first)
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 0 {
		t.Fatalf("first push returned %d messages", len(ready))
	}

	ready, err = m.push(NewTextPart(" world"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 0 {
		t.Fatalf("second push returned %d messages", len(ready))
	}
	if first.Text != "Hello" {
		t.Fatalf("raw message changed to %q", first.Text)
	}

	ready = m.flush()
	if len(ready) != 1 {
		t.Fatalf("flush returned %d messages", len(ready))
	}
	part, ok := ready[0].(*TextPart)
	if !ok || part.Text != "Hello world" {
		t.Fatalf("got %#v, want merged TextPart", ready[0])
	}
}

func TestMergerFlushesContentBeforeTurnEnd(t *testing.T) {
	m := &merger{}
	if _, err := m.push(NewTextPart("answer")); err != nil {
		t.Fatal(err)
	}

	ready, err := m.push(&TurnEnd{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 2 {
		t.Fatalf("got %d messages, want 2", len(ready))
	}
	if part, ok := ready[0].(*TextPart); !ok || part.Text != "answer" {
		t.Fatalf("first message = %#v", ready[0])
	}
	if _, ok := ready[1].(*TurnEnd); !ok {
		t.Fatalf("second message = %T, want *TurnEnd", ready[1])
	}
	if remaining := m.flush(); len(remaining) != 0 {
		t.Fatalf("unexpected remaining messages: %#v", remaining)
	}
}

func TestThinkPartStopsMergingAfterSignature(t *testing.T) {
	m := &merger{}
	signature := "signature"

	if _, err := m.push(NewThinkPart("first ")); err != nil {
		t.Fatal(err)
	}
	signed := NewThinkPart("second")
	signed.Encrypted = &signature
	if _, err := m.push(signed); err != nil {
		t.Fatal(err)
	}

	ready, err := m.push(NewThinkPart("third"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 {
		t.Fatalf("got %d ready messages, want 1", len(ready))
	}
	part, ok := ready[0].(*ThinkPart)
	if !ok || part.Think != "first second" {
		t.Fatalf("ready message = %#v", ready[0])
	}
	if part.Encrypted == nil || *part.Encrypted != signature {
		t.Fatalf("signature was not preserved: %#v", part)
	}
}

func TestMergerBuffersNonConcatenatingContentPart(t *testing.T) {
	var merger merger
	ready, err := merger.push(NewImageURLPart("https://example.com/image.png"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 0 {
		t.Fatalf("got %d ready messages, want 0", len(ready))
	}

	ready, err = merger.push(&TurnEnd{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ready) != 2 {
		t.Fatalf("got %d ready messages, want 2", len(ready))
	}
	if _, ok := ready[0].(*ImageURLPart); !ok {
		t.Fatalf("got %T, want *ImageURLPart", ready[0])
	}
}
