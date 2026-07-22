package wire

import (
	"encoding/json"
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

func TestTurnBeginWithTextRoundTrip(t *testing.T) {
	original := NewTurnBegin(NewTextInput("hello"))

	envelope, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}
	if envelope.Type != "TurnBegin" || string(envelope.Payload) != `{"user_input":"hello"}` {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}

	message, err := Decode(envelope)
	if err != nil {
		t.Fatal(err)
	}
	turn, ok := message.(*TurnBegin)
	if !ok {
		t.Fatalf("got %T, want *TurnBegin", message)
	}
	text, ok := turn.UserInput.Text()
	if !ok || text != "hello" {
		t.Fatalf("got input %q, text=%v", text, ok)
	}
}

func TestTurnBeginWithContentPartsRoundTrip(t *testing.T) {
	original := NewTurnBegin(NewPartsInput(
		NewTextPart("look"),
		NewThinkPart("reason"),
	))

	envelope, err := Encode(original)
	if err != nil {
		t.Fatal(err)
	}

	message, err := Decode(envelope)
	if err != nil {
		t.Fatal(err)
	}
	turn := message.(*TurnBegin)
	parts, ok := turn.UserInput.Parts()
	if !ok || len(parts) != 2 {
		t.Fatalf("got %d parts, parts=%v", len(parts), ok)
	}
	if text, ok := parts[0].(*TextPart); !ok || text.Text != "look" {
		t.Fatalf("first part = %#v", parts[0])
	}
	if think, ok := parts[1].(*ThinkPart); !ok || think.Think != "reason" {
		t.Fatalf("second part = %#v", parts[1])
	}
}

func TestTurnBeginRejectsMissingUserInput(t *testing.T) {
	_, err := Decode(Envelope{
		Type:    "TurnBegin",
		Payload: json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected missing user_input error")
	}
}
