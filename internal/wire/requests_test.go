package wire

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestRequestJSONCompatibility(t *testing.T) {
	arguments := `{"command":"ls"}`
	cases := []struct {
		message Message
		want    string
	}{
		{
			message: &ApprovalResponse{RequestID: "request_123", Response: ApprovalApprove},
			want:    `{"type":"ApprovalResponse","payload":{"request_id":"request_123","response":"approve","feedback":""}}`,
		},
		{
			message: &ApprovalRequest{
				ID: "request_123", ToolCallID: "call_999", Sender: "bash",
				Action: "Execute dangerous command", Description: "This command will delete files",
			},
			want: `{"type":"ApprovalRequest","payload":{"id":"request_123","tool_call_id":"call_999","sender":"bash","action":"Execute dangerous command","description":"This command will delete files","source_kind":null,"source_id":null,"agent_id":null,"subagent_type":null,"source_description":null,"display":[]}}`,
		},
		{
			message: &ToolCallRequest{ID: "call_123", Name: "bash", Arguments: &arguments},
			want:    `{"type":"ToolCallRequest","payload":{"id":"call_123","name":"bash","arguments":"{\"command\":\"ls\"}"}}`,
		},
		{
			message: &QuestionRequest{
				ID: "question_001", ToolCallID: "call_456",
				Questions: []QuestionItem{{
					Question: "Which library?", Header: "Library",
					Options: []QuestionOption{
						{Label: "React", Description: "A JS library"},
						{Label: "Vue", Description: "A progressive framework"},
					},
				}},
			},
			want: `{"type":"QuestionRequest","payload":{"id":"question_001","tool_call_id":"call_456","questions":[{"question":"Which library?","header":"Library","options":[{"label":"React","description":"A JS library"},{"label":"Vue","description":"A progressive framework"}],"multi_select":false,"body":"","other_label":"","other_description":""}]}}`,
		},
		{
			message: &HookRequest{ID: "hook_1", Event: "PreToolUse"},
			want:    `{"type":"HookRequest","payload":{"id":"hook_1","subscription_id":"","event":"PreToolUse","target":"","input_data":{}}}`,
		},
	}

	for _, tc := range cases {
		envelope, err := Encode(tc.message)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, mustMarshal(t, envelope), []byte(tc.want))
		decoded, err := Decode(envelope)
		if err != nil {
			t.Fatal(err)
		}
		roundTrip, err := Encode(decoded)
		if err != nil {
			t.Fatal(err)
		}
		assertJSONEqual(t, mustMarshal(t, roundTrip), []byte(tc.want))
	}
}

func TestApprovalRequestResolveWait(t *testing.T) {
	request := &ApprovalRequest{ID: "approval_1"}
	result := make(chan ApprovalKind, 1)
	go func() {
		response, err := request.Wait(context.Background())
		if err == nil {
			result <- response
		}
	}()

	if request.Resolved() {
		t.Fatal("new request should not be resolved")
	}
	if !request.Resolve(ApprovalReject, "use a safer command") {
		t.Fatal("first resolve should succeed")
	}
	if request.Resolve(ApprovalApprove, "") {
		t.Fatal("second resolve should be ignored")
	}
	if response := <-result; response != ApprovalReject {
		t.Fatalf("got %q", response)
	}
	if request.Feedback() != "use a safer command" {
		t.Fatalf("got feedback %q", request.Feedback())
	}
}

func TestQuestionRequestRejectAndContextCancellation(t *testing.T) {
	rejected := &QuestionRequest{ID: "question_1"}
	if !rejected.Reject(nil) {
		t.Fatal("reject should succeed")
	}
	if _, err := rejected.Wait(context.Background()); !errors.Is(err, ErrQuestionNotSupported) {
		t.Fatalf("got %v", err)
	}

	pending := &QuestionRequest{ID: "question_2"}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := pending.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestWireDeliversResolvableRequest(t *testing.T) {
	w := New()
	subscription, err := w.SubscribeRaw(1)
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Close()

	request := &QuestionRequest{ID: "question_1"}
	if err := w.Send(request); err != nil {
		t.Fatal(err)
	}
	received := receive(t, subscription)
	delivered, ok := received.(*QuestionRequest)
	if !ok || delivered != request {
		t.Fatalf("got %#v, want original request", received)
	}
	if !delivered.Resolve(map[string]string{"Pick?": "A"}) {
		t.Fatal("resolve should succeed")
	}
	answers, err := request.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if answers["Pick?"] != "A" {
		t.Fatalf("got answers %#v", answers)
	}
}

func TestHookRequestResolveWait(t *testing.T) {
	request := &HookRequest{ID: "hook_1"}
	if !request.Resolve(HookBlock, "policy denied") {
		t.Fatal("resolve should succeed")
	}
	response, err := request.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if response.RequestID != "hook_1" || response.Action != HookBlock || response.Reason != "policy denied" {
		t.Fatalf("unexpected hook response: %#v", response)
	}
}

func TestApprovalResponseRejectsInvalidKind(t *testing.T) {
	_, err := Decode(Envelope{
		Type:    "ApprovalResponse",
		Payload: json.RawMessage(`{"request_id":"r1","response":"invalid","feedback":""}`),
	})
	if err == nil {
		t.Fatal("expected invalid approval kind error")
	}
}

func TestLegacyApprovalResponseName(t *testing.T) {
	message, err := Decode(Envelope{
		Type:    "ApprovalRequestResolved",
		Payload: json.RawMessage(`{"request_id":"r1","response":"approve","feedback":""}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := message.(*ApprovalResponse); !ok {
		t.Fatalf("got %T", message)
	}
}
