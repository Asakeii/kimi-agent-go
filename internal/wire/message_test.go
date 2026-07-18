package wire

import (
	"encoding/json"
	"testing"
)

func TestTextPartMarshalJSON(t *testing.T) {
	data, err := json.Marshal(NewTextPart("hello"))
	if err != nil {
		t.Fatalf("Failed to marshal TextPart: %v", err)
	}

	want := `{"type":"text","text":"hello"}`
	if string(data) != want {
		t.Errorf("MarshalJSON() = %s; want %s", data, want)
	}
}

func TestThinkPartMarshalJSON(t *testing.T) {
	data, err := json.Marshal(NewThinkPart("checking files"))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"type":"think","think":"checking files","encrypted":null}`
	if string(data) != want {
		t.Fatalf("got %s, want %s", data, want)
	}
}
