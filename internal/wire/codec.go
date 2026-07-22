package wire

import (
	"encoding/json"
	"fmt"
)

type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Encode 将 Message 编码为 Envelope
func Encode(message Message) (Envelope, error) {
	if message == nil {
		return Envelope{}, fmt.Errorf("wire: cannot encode nil message")
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

// Decode 将 Envelope 解码为 Message
func Decode(envelope Envelope) (Message, error) {
	switch envelope.Type {
	case "ContentPart":
		return decodeContentPart(envelope.Payload)
	case "TurnEnd":
		var event TurnEnd
		if err := json.Unmarshal(envelope.Payload, &event); err != nil {
			return nil, fmt.Errorf("wire: failed to unmarshal TurnEnd: %w", err)
		}
		return &event, nil
	case "":
		return nil, fmt.Errorf("wire: envelope type is empty")
	default:
		return nil, fmt.Errorf("wire: unknown envelope type: %s", envelope.Type)
	}
}

// decodeContentPart 解码 ContentPart
func decodeContentPart(payload json.RawMessage) (ContentPart, error) {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &header); err != nil {
		return nil, fmt.Errorf("wire: failed to unmarshal ContentPart header: %w", err)
	}

	switch header.Type {
	case "text":
		var textPart TextPart
		if err := json.Unmarshal(payload, &textPart); err != nil {
			return nil, fmt.Errorf("wire: failed to unmarshal TextPart: %w", err)
		}
		return &textPart, nil
	case "think":
		var thinkPart ThinkPart
		if err := json.Unmarshal(payload, &thinkPart); err != nil {
			return nil, fmt.Errorf("wire: failed to unmarshal ThinkPart: %w", err)
		}
		return &thinkPart, nil
	case "":
		return nil, fmt.Errorf("wire: ContentPart type is empty")
	default:
		return nil, fmt.Errorf("wire: unknown ContentPart type: %s", header.Type)
	}
}
