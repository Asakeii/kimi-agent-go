package wire

import (
	"reflect"
	"sort"
	"testing"
)

func TestProtocolRegistryMatchesPythonWire110(t *testing.T) {
	eventTypes := []string{
		"ApprovalResponse",
		"BtwBegin",
		"BtwEnd",
		"CompactionBegin",
		"CompactionEnd",
		"HookResolved",
		"HookTriggered",
		"MCPLoadingBegin",
		"MCPLoadingEnd",
		"Notification",
		"PlanDisplay",
		"StatusUpdate",
		"SteerInput",
		"StepBegin",
		"StepInterrupted",
		"StepRetry",
		"SubagentEvent",
		"ToolCall",
		"ToolCallPart",
		"ToolResult",
		"TurnBegin",
		"TurnEnd",
	}
	requestTypes := []string{
		"ApprovalRequest",
		"HookRequest",
		"QuestionRequest",
		"ToolCallRequest",
	}

	for _, name := range eventTypes {
		factory, ok := messageFactories[name]
		if !ok {
			t.Errorf("event %s is not registered", name)
			continue
		}
		if _, ok := factory().(Event); !ok {
			t.Errorf("%s factory returned %T, want Event", name, factory())
		}
	}
	for _, name := range requestTypes {
		factory, ok := messageFactories[name]
		if !ok {
			t.Errorf("request %s is not registered", name)
			continue
		}
		if _, ok := factory().(Request); !ok {
			t.Errorf("%s factory returned %T, want Request", name, factory())
		}
	}

	wantNames := append(append([]string{}, eventTypes...), requestTypes...)
	wantNames = append(wantNames, "ApprovalRequestResolved")
	sort.Strings(wantNames)
	gotNames := make([]string, 0, len(messageFactories))
	for name := range messageFactories {
		gotNames = append(gotNames, name)
	}
	sort.Strings(gotNames)
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("registered message types:\n got %v\nwant %v", gotNames, wantNames)
	}

	wantContent := []string{"audio_url", "image_url", "text", "think", "video_url"}
	gotContent := make([]string, 0, len(contentPartFactories))
	for name, factory := range contentPartFactories {
		gotContent = append(gotContent, name)
		part := factory()
		if _, ok := part.(Mergeable); !ok {
			t.Errorf("content part %s does not implement Mergeable", name)
		}
	}
	sort.Strings(gotContent)
	if !reflect.DeepEqual(gotContent, wantContent) {
		t.Fatalf("registered content types: got %v, want %v", gotContent, wantContent)
	}
}

func TestEncodeRejectsTypedNilMessage(t *testing.T) {
	var part *TextPart
	if _, err := Encode(part); err == nil {
		t.Fatal("expected typed nil message error")
	}
	if err := New().Send(part); err == nil {
		t.Fatal("expected typed nil send error")
	}
}

func TestOnlyPointerMessagesImplementProtocol(t *testing.T) {
	if _, ok := any(TextPart{}).(Message); ok {
		t.Fatal("TextPart value unexpectedly implements Message")
	}
	if _, ok := any(TurnEnd{}).(Message); ok {
		t.Fatal("TurnEnd value unexpectedly implements Message")
	}
}
