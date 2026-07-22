package wire

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestControlAndStatusEventJSONCompatibility(t *testing.T) {
	statusCode := 429
	contextUsage := 0.5
	cases := []struct {
		name    string
		message Message
		want    string
	}{
		{
			name:    "steer input",
			message: NewSteerInput(NewTextInput("Follow up")),
			want:    `{"type":"SteerInput","payload":{"user_input":"Follow up"}}`,
		},
		{
			name:    "step begin",
			message: &StepBegin{N: 1},
			want:    `{"type":"StepBegin","payload":{"n":1}}`,
		},
		{
			name:    "hook defaults",
			message: &HookTriggered{Event: "PreToolUse"},
			want:    `{"type":"HookTriggered","payload":{"event":"PreToolUse","target":"","hook_count":1}}`,
		},
		{
			name:    "hook resolution defaults",
			message: &HookResolved{Event: "PreToolUse"},
			want:    `{"type":"HookResolved","payload":{"event":"PreToolUse","target":"","action":"allow","reason":"","duration_ms":0}}`,
		},
		{
			name: "step retry",
			message: &StepRetry{
				N: 1, NextAttempt: 2, MaxAttempts: 3, WaitSeconds: 1.25,
				ErrorType: "APIStatusError", StatusCode: &statusCode,
			},
			want: `{"type":"StepRetry","payload":{"n":1,"next_attempt":2,"max_attempts":3,"wait_s":1.25,"error_type":"APIStatusError","status_code":429}}`,
		},
		{
			name: "status update",
			message: &StatusUpdate{
				ContextUsage: &contextUsage,
				MCPStatus: &MCPStatusSnapshot{
					Loading: true, Total: 1,
					Servers: []MCPServerSnapshot{{Name: "context7", Status: "connecting"}},
				},
			},
			want: `{"type":"StatusUpdate","payload":{"context_usage":0.5,"context_tokens":null,"max_context_tokens":null,"token_usage":null,"message_id":null,"plan_mode":null,"mcp_status":{"loading":true,"connected":0,"total":1,"tools":0,"servers":[{"name":"context7","status":"connecting","tools":[]}]}}}`,
		},
		{
			name: "notification defaults",
			message: &Notification{
				ID: "n1", Category: "task", Type: "task.completed", SourceKind: "task",
				SourceID: "b1", Title: "Done", Body: "Finished", Severity: "success", CreatedAt: 123.456,
			},
			want: `{"type":"Notification","payload":{"id":"n1","category":"task","type":"task.completed","source_kind":"task","source_id":"b1","title":"Done","body":"Finished","severity":"success","created_at":123.456,"payload":{}}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			envelope, err := Encode(tc.message)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, data, []byte(tc.want))

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
		})
	}
}

func TestSimpleControlEventsRoundTrip(t *testing.T) {
	messages := []Message{
		&TurnEnd{},
		&StepInterrupted{},
		&CompactionBegin{},
		&CompactionEnd{},
		&MCPLoadingBegin{},
		&MCPLoadingEnd{},
		&PlanDisplay{Content: "## Plan", FilePath: "/tmp/plan.md"},
		&BtwBegin{ID: "btw-1", Question: "why?"},
		&BtwEnd{ID: "btw-1"},
	}

	for _, message := range messages {
		envelope, err := Encode(message)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := Decode(envelope)
		if err != nil {
			t.Fatalf("decode %s: %v", envelope.Type, err)
		}
		if reflect.TypeOf(decoded) != reflect.TypeOf(message) {
			t.Fatalf("got %T, want %T", decoded, message)
		}
	}
}

func TestMediaContentPartsRoundTrip(t *testing.T) {
	id := "clip"
	cases := []struct {
		part ContentPart
		want string
	}{
		{NewImageURLPart("https://example.com/image.png"), `{"type":"image_url","image_url":{"url":"https://example.com/image.png","id":null}}`},
		{&AudioURLPart{AudioURL: MediaURL{URL: "https://example.com/audio", ID: &id}}, `{"type":"audio_url","audio_url":{"url":"https://example.com/audio","id":"clip"}}`},
		{NewVideoURLPart("https://example.com/video"), `{"type":"video_url","video_url":{"url":"https://example.com/video","id":null}}`},
	}

	for _, tc := range cases {
		envelope, err := Encode(tc.part)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, envelope.Payload, []byte(tc.want))
		decoded, err := Decode(envelope)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.TypeOf(decoded) != reflect.TypeOf(tc.part) {
			t.Fatalf("got %T, want %T", decoded, tc.part)
		}
	}
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("invalid got JSON %s: %v", got, err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("invalid want JSON %s: %v", want, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("got %s, want %s", got, want)
	}
}
