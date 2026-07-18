package wire

/*
wire 是 kimi-agent-go 内部的消息传递协议，主要用于模型与UI之间的通信。
Message 是所有 wire 消息的基类
*/
import (
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
	isContentpart()
}

// TextPart 支持模型输出的 chunk，TextPart是ContentPart的一种实现
type TextPart struct {
	Text string
}

type ThinkPart struct {
	Text      string
	Encrypted *string // 表示是否加密
}

// NewTextPart 创建一个新的 TextPart 实例
func NewTextPart(text string) *TextPart {
	return &TextPart{
		Text: text,
	}
}

func NewThinkPart(text string) *ThinkPart {
	return &ThinkPart{
		Text: text,
	}
}

func (t *TextPart) wireType() string {
	return "ContentPart"
}
func (t *TextPart) isMessage()     {}
func (t *TextPart) isEvent()       {}
func (t *TextPart) isContentpart() {}

func (t *ThinkPart) wireType() string {
	return "ContentPart"
}
func (t *ThinkPart) isMessage()     {}
func (t *ThinkPart) isEvent()       {}
func (t *ThinkPart) isContentpart() {}

// 编译期检查
var _ ContentPart = (*TextPart)(nil)
var _ ContentPart = (*ThinkPart)(nil)

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
		Text      string  `json:"text"`
		Encrypted *string `json:"encrypted,omitempty"`
	}{
		Type:      "think",
		Text:      t.Text,
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
		Text      *string `json:"text"`
		Encrypted *string `json:"encrypted,omitempty"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	if payload.Text == nil {
		return fmt.Errorf("think part is missing text")
	}

	if payload.Type != "think" {
		return fmt.Errorf("invalid type for ThinkPart: %s", payload.Type)
	}

	t.Text = *payload.Text
	t.Encrypted = payload.Encrypted

	return nil
}
