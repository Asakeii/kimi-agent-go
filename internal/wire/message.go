package wire

/*
wire 是 kimi-agent-go 内部的消息传递协议，主要用于模型与UI之间的通信。
Message 是所有 wire 消息的基类
*/
import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Message wire消息基类
type Message interface {
	wireType() string
	isMessage()
}

// Event wire事件，用于通知UI，只通知不回复
type Event interface {
	Message
	isEvent()
}

// Request wire请求，需要等待通知
type Request interface {
	Message
	RequestID() string
	isRequest()
}

// ContentPart Event 的内容部分，主要用于模型向 UI 输出的内容事件
type ContentPart interface {
	Event
	isContentPart()
}

type Mergeable interface {
	Message
	Clone() Mergeable                 // 复制一份作为 merged 缓冲，防止修改 raw 消息
	MergeInPlace(next Mergeable) bool // 尝试把下一块合进当前对象，成功返回 true
}

// TextPart 支持模型输出的 chunk，TextPart是ContentPart的一种实现
type TextPart struct {
	Text string
}

type ThinkPart struct {
	Think     string
	Encrypted *string // 模型返回的加密 reasoning signature
}

type userInputKind uint8

const (
	userInputInvalid userInputKind = iota
	userInputText
	userInputParts
)

// UserInput 对应 Python 的 str | list[ContentPart] 联合类型。
type UserInput struct {
	kind  userInputKind
	text  string
	parts []ContentPart
}

func NewTextInput(text string) UserInput {
	return UserInput{kind: userInputText, text: text}
}

func NewPartsInput(parts ...ContentPart) UserInput {
	copied := make([]ContentPart, len(parts))
	copy(copied, parts)
	return UserInput{kind: userInputParts, parts: copied}
}

func (u UserInput) Text() (string, bool) {
	return u.text, u.kind == userInputText
}

func (u UserInput) Parts() ([]ContentPart, bool) {
	if u.kind != userInputParts {
		return nil, false
	}
	copied := make([]ContentPart, len(u.parts))
	copy(copied, u.parts)
	return copied, true
}

func (u UserInput) MarshalJSON() ([]byte, error) {
	switch u.kind {
	case userInputText:
		return json.Marshal(u.text)
	case userInputParts:
		for i, part := range u.parts {
			if part == nil {
				return nil, fmt.Errorf("wire: user input part %d is nil", i)
			}
		}
		return json.Marshal(u.parts)
	default:
		return nil, fmt.Errorf("wire: user input is not initialized")
	}
}

func (u *UserInput) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return fmt.Errorf("wire: user input is empty")
	}

	switch data[0] {
	case '"':
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return fmt.Errorf("wire: decode text input: %w", err)
		}
		*u = NewTextInput(text)
		return nil
	case '[':
		var rawParts []json.RawMessage
		if err := json.Unmarshal(data, &rawParts); err != nil {
			return fmt.Errorf("wire: decode content parts: %w", err)
		}
		parts := make([]ContentPart, 0, len(rawParts))
		for i, rawPart := range rawParts {
			part, err := decodeContentPart(rawPart)
			if err != nil {
				return fmt.Errorf("wire: decode user input part %d: %w", i, err)
			}
			parts = append(parts, part)
		}
		*u = NewPartsInput(parts...)
		return nil
	default:
		return fmt.Errorf("wire: user input must be a string or content-part array")
	}
}

type TurnBegin struct {
	UserInput UserInput `json:"user_input"`
}

func NewTurnBegin(userInput UserInput) *TurnBegin {
	return &TurnBegin{UserInput: userInput}
}

func (*TurnBegin) wireType() string { return "TurnBegin" }
func (*TurnBegin) isMessage()       {}
func (*TurnBegin) isEvent()         {}

func (t *TurnBegin) UnmarshalJSON(data []byte) error {
	var payload struct {
		UserInput json.RawMessage `json:"user_input"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if len(payload.UserInput) == 0 {
		return fmt.Errorf("wire: TurnBegin is missing user_input")
	}
	return json.Unmarshal(payload.UserInput, &t.UserInput)
}

var _ Event = (*TurnBegin)(nil)

type TurnEnd struct{}

func (*TurnEnd) wireType() string {
	return "TurnEnd"
}

func (*TurnEnd) isMessage() {}

func (*TurnEnd) isEvent() {}

var _ Event = (*TurnEnd)(nil)

// NewTextPart 创建一个新的 TextPart 实例
func NewTextPart(text string) *TextPart {
	return &TextPart{
		Text: text,
	}
}

// NewThinkPart 创建一个新的 ThinkPart 实例
func NewThinkPart(think string) *ThinkPart {
	return &ThinkPart{
		Think: think,
	}
}

// Clone 负责创建不影响 raw 消息的合并副本，副本将用于 MergeInPlace 方法中，防止修改原始消息，原始消息会被用于流式输出
func (t *TextPart) Clone() Mergeable {
	return &TextPart{
		Text: t.Text,
	}
}

// MergeInPlace 负责把后续流式碎片不断追加到这个副本上，流结束后副本用于保存完整消息
func (t *TextPart) MergeInPlace(next Mergeable) bool {
	other, ok := next.(*TextPart) // 类型断言，确保 next 是 TextPart 类型
	if !ok {
		return false
	}

	t.Text += other.Text
	return true
}

// Clone 创建独立的思考块副本，避免 merged 缓冲修改 raw 消息。
func (t *ThinkPart) Clone() Mergeable {
	cloned := &ThinkPart{Think: t.Think}
	if t.Encrypted != nil {
		encrypted := *t.Encrypted
		cloned.Encrypted = &encrypted
	}
	return cloned
}

// MergeInPlace 合并连续的思考块。已有非空签名时不能继续追加内容。
func (t *ThinkPart) MergeInPlace(next Mergeable) bool {
	other, ok := next.(*ThinkPart)
	if !ok {
		return false
	}
	if t.Encrypted != nil && *t.Encrypted != "" {
		return false
	}

	t.Think += other.Think
	if other.Encrypted != nil && *other.Encrypted != "" {
		encrypted := *other.Encrypted
		t.Encrypted = &encrypted
	}
	return true
}

func (t *TextPart) wireType() string {
	return "ContentPart"
}
func (t *TextPart) isMessage()     {}
func (t *TextPart) isEvent()       {}
func (t *TextPart) isContentPart() {}

func (t *ThinkPart) wireType() string {
	return "ContentPart"
}
func (t *ThinkPart) isMessage()     {}
func (t *ThinkPart) isEvent()       {}
func (t *ThinkPart) isContentPart() {}

// 编译期检查
var _ ContentPart = (*TextPart)(nil)
var _ ContentPart = (*ThinkPart)(nil)
var _ Mergeable = (*TextPart)(nil)
var _ Mergeable = (*ThinkPart)(nil)

// MarshalJSON 实现了自定义的 JSON 序列化方法，将 TextPart 序列化为 JSON 格式
func (t *TextPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{
		Type: "text",
		Text: t.Text,
	})
}

// MarshalJSON 实现了自定义的 JSON 序列化方法，将 ThinkPart 序列化为 JSON 格式
func (t *ThinkPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      string  `json:"type"`
		Think     string  `json:"think"`
		Encrypted *string `json:"encrypted"`
	}{
		Type:      "think",
		Think:     t.Think,
		Encrypted: t.Encrypted,
	})
}

// UnmarshalJSON 实现了自定义的 JSON 反序列化方法，将 JSON 格式的数据反序列化为 TextPart 对象
func (t *TextPart) UnmarshalJSON(data []byte) error {
	var payload struct {
		Type string  `json:"type"`
		Text *string `json:"text"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if payload.Text == nil {
		return fmt.Errorf("text part is missing text")
	}

	if payload.Type != "text" {
		return fmt.Errorf("invalid type for TextPart: %s", payload.Type)
	}

	t.Text = *payload.Text

	return nil
}

// UnmarshalJSON 实现了自定义的 JSON 反序列化方法，将 JSON 格式的数据反序列化为 ThinkPart 对象
func (t *ThinkPart) UnmarshalJSON(data []byte) error {
	var payload struct {
		Type      string  `json:"type"`
		Think     *string `json:"think"`
		Encrypted *string `json:"encrypted"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if payload.Type != "think" {
		return fmt.Errorf("invalid type for ThinkPart: %s", payload.Type)
	}
	if payload.Think == nil {
		return fmt.Errorf("think part is missing think")
	}

	t.Think = *payload.Think
	t.Encrypted = payload.Encrypted

	return nil
}
