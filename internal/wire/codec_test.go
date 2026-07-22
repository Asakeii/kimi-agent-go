package wire

import (
	"testing"
)

func TestTextPartRoundTrip(t *testing.T) {
	original := NewTextPart("hello")

	envelope, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}

	message, err := Decode(envelope)
	if err != nil {
		t.Fatal(err)
	}

	part, ok := message.(*TextPart)
	if !ok {
		t.Fatalf("got %T, want *TextPart", message)
	}

	if part.Text != "hello" {
		t.Fatalf("got %q, want %q", part.Text, "hello")
	}
}

func TestThinkPartRoundTrip(t *testing.T) {
	encrypted := "signature"
	original := NewThinkPart("checking files")
	original.Encrypted = &encrypted

	envelope, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}

	message, err := Decode(envelope)
	if err != nil {
		t.Fatal(err)
	}

	part, ok := message.(*ThinkPart)
	if !ok {
		t.Fatalf("got %T, want *ThinkPart", message)
	}
	if part.Think != "checking files" {
		t.Fatalf("got think %q", part.Think)
	}
	if part.Encrypted == nil || *part.Encrypted != encrypted {
		t.Fatalf("got encrypted %v", part.Encrypted)
	}
}

func TestTurnEndRoundTrip(t *testing.T) {
	envelope, err := Encode(&TurnEnd{})
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Type != "TurnEnd" || string(envelope.Payload) != "{}" {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}

	message, err := Decode(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := message.(*TurnEnd); !ok {
		t.Fatalf("got %T, want *TurnEnd", message)
	}
}
