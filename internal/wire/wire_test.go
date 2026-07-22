package wire

import (
	"errors"
	"io"
	"testing"
	"time"
)

func TestWirePublishesRawAndMergedMessages(t *testing.T) {
	w := New()
	raw, err := w.SubscribeRaw(1)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	merged, err := w.SubscribeMerged(1)
	if err != nil {
		t.Fatal(err)
	}
	defer merged.Close()

	for _, message := range []Message{
		NewTextPart("Hello"),
		NewTextPart(" world"),
		&TurnEnd{},
	} {
		if err := w.Send(message); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	assertTextMessage(t, receive(t, raw), "Hello")
	assertTextMessage(t, receive(t, raw), " world")
	assertTurnEnd(t, receive(t, raw))
	assertEOF(t, raw)

	assertTextMessage(t, receive(t, merged), "Hello world")
	assertTurnEnd(t, receive(t, merged))
	assertEOF(t, merged)
}

func TestSlowSubscriberDoesNotBlockSend(t *testing.T) {
	w := New()
	raw, err := w.SubscribeRaw(0)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()

	done := make(chan error, 1)
	go func() {
		for i := 0; i < 1_000; i++ {
			if err := w.Send(NewTextPart("chunk")); err != nil {
				done <- err
				return
			}
		}
		done <- w.Close()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("slow subscriber blocked Wire.Send")
	}
}

func TestSubscribeAfterClose(t *testing.T) {
	w := New()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := w.SubscribeRaw(1); !errors.Is(err, ErrClosed) {
		t.Fatalf("got %v, want ErrClosed", err)
	}
}

func receive(t *testing.T, subscription *Subscription) Message {
	t.Helper()
	message, err := subscription.Recv()
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func assertTextMessage(t *testing.T, message Message, want string) {
	t.Helper()
	part, ok := message.(*TextPart)
	if !ok || part.Text != want {
		t.Fatalf("got %#v, want TextPart(%q)", message, want)
	}
}

func assertTurnEnd(t *testing.T, message Message) {
	t.Helper()
	if _, ok := message.(*TurnEnd); !ok {
		t.Fatalf("got %T, want *TurnEnd", message)
	}
}

func assertEOF(t *testing.T, subscription *Subscription) {
	t.Helper()
	if _, err := subscription.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("got %v, want EOF", err)
	}
}
