package wire

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Encode 将 Message 编码为 Envelope
func Encode(message Message) (Envelope, error) {
	if isNilMessage(message) {
		return Envelope{}, fmt.Errorf("wire: cannot encode nil message")
	}
	if validator, ok := message.(interface{ validate() error }); ok {
		if err := validator.validate(); err != nil {
			return Envelope{}, err
		}
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return Envelope{}, fmt.Errorf("wire: failed to marshal message: %w", err)
	}

	return Envelope{
		Type:    message.wireType(),
		Payload: payload,
	}, nil
}

func isNilMessage(message Message) bool {
	if message == nil {
		return true
	}
	value := reflect.ValueOf(message)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// Decode 将 Envelope 解码为 Message
func Decode(envelope Envelope) (Message, error) {
	if envelope.Type == "ContentPart" {
		return decodeContentPart(envelope.Payload)
	}
	if envelope.Type == "" {
		return nil, fmt.Errorf("wire: envelope type is empty")
	}

	factory, ok := messageFactories[envelope.Type]
	if !ok {
		return nil, fmt.Errorf("wire: unknown envelope type: %s", envelope.Type)
	}
	message := factory()
	if err := json.Unmarshal(envelope.Payload, message); err != nil {
		return nil, fmt.Errorf("wire: failed to unmarshal %s: %w", envelope.Type, err)
	}
	if validator, ok := message.(interface{ validate() error }); ok {
		if err := validator.validate(); err != nil {
			return nil, err
		}
	}
	return message, nil
}

// decodeContentPart 解码 ContentPart
func decodeContentPart(payload json.RawMessage) (ContentPart, error) {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &header); err != nil {
		return nil, fmt.Errorf("wire: failed to unmarshal ContentPart header: %w", err)
	}

	if header.Type == "" {
		return nil, fmt.Errorf("wire: ContentPart type is empty")
	}
	factory, ok := contentPartFactories[header.Type]
	if !ok {
		return nil, fmt.Errorf("wire: unknown ContentPart type: %s", header.Type)
	}
	part := factory()
	if err := json.Unmarshal(payload, part); err != nil {
		return nil, fmt.Errorf("wire: failed to unmarshal %s ContentPart: %w", header.Type, err)
	}
	return part, nil
}

var messageFactories = map[string]func() Message{
	"TurnBegin":        func() Message { return &TurnBegin{} },
	"SteerInput":       func() Message { return &SteerInput{} },
	"TurnEnd":          func() Message { return &TurnEnd{} },
	"StepBegin":        func() Message { return &StepBegin{} },
	"StepInterrupted":  func() Message { return &StepInterrupted{} },
	"StepRetry":        func() Message { return &StepRetry{} },
	"CompactionBegin":  func() Message { return &CompactionBegin{} },
	"CompactionEnd":    func() Message { return &CompactionEnd{} },
	"HookTriggered":    func() Message { return &HookTriggered{HookCount: 1} },
	"HookResolved":     func() Message { return &HookResolved{Action: "allow"} },
	"MCPLoadingBegin":  func() Message { return &MCPLoadingBegin{} },
	"MCPLoadingEnd":    func() Message { return &MCPLoadingEnd{} },
	"StatusUpdate":     func() Message { return &StatusUpdate{} },
	"Notification":     func() Message { return &Notification{Payload: map[string]any{}} },
	"PlanDisplay":      func() Message { return &PlanDisplay{} },
	"BtwBegin":         func() Message { return &BtwBegin{} },
	"BtwEnd":           func() Message { return &BtwEnd{} },
	"ToolCall":         func() Message { return &ToolCall{} },
	"ToolCallPart":     func() Message { return &ToolCallPart{} },
	"ToolResult":       func() Message { return &ToolResult{} },
	"SubagentEvent":    func() Message { return &SubagentEvent{} },
	"ApprovalResponse": func() Message { return &ApprovalResponse{} },
	"ApprovalRequest":  func() Message { return &ApprovalRequest{} },
	"QuestionRequest":  func() Message { return &QuestionRequest{} },
	"ToolCallRequest":  func() Message { return &ToolCallRequest{} },
	"HookRequest":      func() Message { return &HookRequest{InputData: map[string]any{}} },
	// Wire v1 used this name for ApprovalResponse.
	"ApprovalRequestResolved": func() Message { return &ApprovalResponse{} },
}

var contentPartFactories = map[string]func() ContentPart{
	"text":      func() ContentPart { return &TextPart{} },
	"think":     func() ContentPart { return &ThinkPart{} },
	"image_url": func() ContentPart { return &ImageURLPart{} },
	"audio_url": func() ContentPart { return &AudioURLPart{} },
	"video_url": func() ContentPart { return &VideoURLPart{} },
}
