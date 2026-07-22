package wire

import (
	"encoding/json"
	"testing"
)

func TestToolEventsJSONCompatibility(t *testing.T) {
	arguments := `{"command": "ls -la"}`
	argumentPart := "}"
	cases := []struct {
		message Message
		want    string
	}{
		{
			message: &ToolCall{ID: "call_123", Function: ToolFunction{Name: "bash", Arguments: &arguments}},
			want:    `{"type":"ToolCall","payload":{"type":"function","id":"call_123","function":{"name":"bash","arguments":"{\"command\": \"ls -la\"}"},"extras":null}}`,
		},
		{
			message: &ToolCallPart{ArgumentsPart: &argumentPart},
			want:    `{"type":"ToolCallPart","payload":{"arguments_part":"}"}}`,
		},
		{
			message: &ToolResult{
				ToolCallID: "call_123",
				ReturnValue: ToolReturnValue{
					Output:  NewTextToolOutput(""),
					Message: "Command completed",
					Display: []DisplayBlock{&BriefDisplayBlock{Text: "Command completed"}},
				},
			},
			want: `{"type":"ToolResult","payload":{"tool_call_id":"call_123","return_value":{"is_error":false,"output":"","message":"Command completed","display":[{"type":"brief","text":"Command completed"}],"extras":null}}}`,
		},
	}

	for _, tc := range cases {
		envelope, err := Encode(tc.message)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, encoded, []byte(tc.want))

		decoded, err := Decode(envelope)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := Encode(decoded)
		if err != nil {
			t.Fatal(err)
		}
		roundTripJSON, err := json.Marshal(roundTrip)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, roundTripJSON, []byte(tc.want))
	}
}

func TestToolCallMergesArgumentParts(t *testing.T) {
	arguments := `{"path":"file.txt"`
	call := &ToolCall{ID: "call_1", Function: ToolFunction{Name: "read", Arguments: &arguments}}
	buffer := call.Clone()
	closing := "}"
	if !buffer.MergeInPlace(&ToolCallPart{ArgumentsPart: &closing}) {
		t.Fatal("expected tool call part to merge")
	}

	merged := buffer.(*ToolCall)
	if got := *merged.Function.Arguments; got != `{"path":"file.txt"}` {
		t.Fatalf("got arguments %q", got)
	}
	if got := *call.Function.Arguments; got != `{"path":"file.txt"` {
		t.Fatalf("raw tool call was changed to %q", got)
	}
}

func TestDisplayBlocksRoundTrip(t *testing.T) {
	returnValue := ToolReturnValue{
		Output: NewPartsToolOutput(NewTextPart("result")),
		Display: []DisplayBlock{
			&DiffDisplayBlock{Path: "a.go", OldText: "old", NewText: "new"},
			&TodoDisplayBlock{Items: []TodoDisplayItem{{Title: "test", Status: "done"}}},
			&ShellDisplayBlock{Language: "bash", Command: "go test ./..."},
			&BackgroundTaskDisplayBlock{TaskID: "b1", Kind: "shell", Status: "running", Description: "tests"},
		},
	}
	data, err := json.Marshal(returnValue)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ToolReturnValue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Display) != 4 {
		t.Fatalf("got %d display blocks", len(decoded.Display))
	}
	if diff, ok := decoded.Display[0].(*DiffDisplayBlock); !ok || diff.OldStart != 1 || diff.NewStart != 1 {
		t.Fatalf("unexpected diff block: %#v", decoded.Display[0])
	}
	parts, ok := decoded.Output.Parts()
	if !ok || len(parts) != 1 {
		t.Fatalf("unexpected tool output: %#v", decoded.Output)
	}
}

func TestEmptyPartListsEncodeAsJSONArray(t *testing.T) {
	userInput, err := json.Marshal(NewPartsInput())
	if err != nil {
		t.Fatal(err)
	}
	if string(userInput) != "[]" {
		t.Fatalf("user input = %s, want []", userInput)
	}
	toolOutput, err := json.Marshal(NewPartsToolOutput())
	if err != nil {
		t.Fatal(err)
	}
	if string(toolOutput) != "[]" {
		t.Fatalf("tool output = %s, want []", toolOutput)
	}
}

func TestUnknownDisplayBlockFallback(t *testing.T) {
	block, err := decodeDisplayBlock(json.RawMessage(`{"type":"custom","value":42}`))
	if err != nil {
		t.Fatal(err)
	}
	unknown, ok := block.(*UnknownDisplayBlock)
	if !ok || unknown.Type != "custom" {
		t.Fatalf("got %#v", block)
	}
	assertJSONEqual(t, mustMarshal(t, unknown), []byte(`{"type":"custom","data":{"value":42}}`))
}

func TestSubagentEventRoundTripAndLegacyField(t *testing.T) {
	parentID := "call_parent"
	agentID := "a123"
	subagentType := "coder"
	event := &SubagentEvent{
		ParentToolCallID: &parentID,
		AgentID:          &agentID,
		SubagentType:     &subagentType,
		Event:            &StepBegin{N: 2},
	}
	envelope, err := Encode(event)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"type":"SubagentEvent","payload":{"parent_tool_call_id":"call_parent","agent_id":"a123","subagent_type":"coder","event":{"type":"StepBegin","payload":{"n":2}}}}`
	assertJSONEqual(t, mustMarshal(t, envelope), []byte(want))

	legacy := Envelope{
		Type: "SubagentEvent",
		Payload: json.RawMessage(`{
			"task_tool_call_id":"legacy_parent",
			"event":{"type":"StepBegin","payload":{"n":3}}
		}`),
	}
	decoded, err := Decode(legacy)
	if err != nil {
		t.Fatal(err)
	}
	subagent := decoded.(*SubagentEvent)
	if subagent.ParentToolCallID == nil || *subagent.ParentToolCallID != "legacy_parent" {
		t.Fatalf("legacy parent id was not restored: %#v", subagent.ParentToolCallID)
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
