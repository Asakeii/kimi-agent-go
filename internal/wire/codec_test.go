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
