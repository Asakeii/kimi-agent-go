package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type TextPart struct {
	contentPartMarker
	Text string `json:"text"`
}

type ThinkPart struct {
	contentPartMarker
	Think     string  `json:"think"`
	Encrypted *string `json:"encrypted"`
}

type MediaURL struct {
	URL string  `json:"url"`
	ID  *string `json:"id"`
}

type ImageURLPart struct {
	contentPartMarker
	ImageURL MediaURL `json:"image_url"`
}

type AudioURLPart struct {
	contentPartMarker
	AudioURL MediaURL `json:"audio_url"`
}

type VideoURLPart struct {
	contentPartMarker
	VideoURL MediaURL `json:"video_url"`
}

func NewTextPart(text string) *TextPart {
	return &TextPart{Text: text}
}

func NewThinkPart(think string) *ThinkPart {
	return &ThinkPart{Think: think}
}

func NewImageURLPart(url string) *ImageURLPart {
	return &ImageURLPart{ImageURL: MediaURL{URL: url}}
}

func NewAudioURLPart(url string) *AudioURLPart {
	return &AudioURLPart{AudioURL: MediaURL{URL: url}}
}

func NewVideoURLPart(url string) *VideoURLPart {
	return &VideoURLPart{VideoURL: MediaURL{URL: url}}
}

func (p *TextPart) Clone() Mergeable {
	return &TextPart{Text: p.Text}
}

func (p *TextPart) MergeInPlace(next Mergeable) bool {
	other, ok := next.(*TextPart)
	if !ok {
		return false
	}
	p.Text += other.Text
	return true
}

func (p *ThinkPart) Clone() Mergeable {
	cloned := &ThinkPart{Think: p.Think}
	if p.Encrypted != nil {
		encrypted := *p.Encrypted
		cloned.Encrypted = &encrypted
	}
	return cloned
}

func (p *ThinkPart) MergeInPlace(next Mergeable) bool {
	other, ok := next.(*ThinkPart)
	if !ok || p.Encrypted != nil && *p.Encrypted != "" {
		return false
	}
	p.Think += other.Think
	if other.Encrypted != nil && *other.Encrypted != "" {
		encrypted := *other.Encrypted
		p.Encrypted = &encrypted
	}
	return true
}

func (p *ImageURLPart) Clone() Mergeable {
	return &ImageURLPart{ImageURL: cloneMediaURL(p.ImageURL)}
}
func (*ImageURLPart) MergeInPlace(Mergeable) bool { return false }
func (p *AudioURLPart) Clone() Mergeable {
	return &AudioURLPart{AudioURL: cloneMediaURL(p.AudioURL)}
}
func (*AudioURLPart) MergeInPlace(Mergeable) bool { return false }
func (p *VideoURLPart) Clone() Mergeable {
	return &VideoURLPart{VideoURL: cloneMediaURL(p.VideoURL)}
}
func (*VideoURLPart) MergeInPlace(Mergeable) bool { return false }

func cloneMediaURL(mediaURL MediaURL) MediaURL {
	cloned := MediaURL{URL: mediaURL.URL}
	if mediaURL.ID != nil {
		id := *mediaURL.ID
		cloned.ID = &id
	}
	return cloned
}

func (p TextPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "text", Text: p.Text})
}

func (p ThinkPart) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type      string  `json:"type"`
		Think     string  `json:"think"`
		Encrypted *string `json:"encrypted"`
	}{Type: "think", Think: p.Think, Encrypted: p.Encrypted})
}

func (p ImageURLPart) MarshalJSON() ([]byte, error) {
	return marshalMediaPart("image_url", "image_url", p.ImageURL)
}

func (p AudioURLPart) MarshalJSON() ([]byte, error) {
	return marshalMediaPart("audio_url", "audio_url", p.AudioURL)
}

func (p VideoURLPart) MarshalJSON() ([]byte, error) {
	return marshalMediaPart("video_url", "video_url", p.VideoURL)
}

func marshalMediaPart(partType, field string, mediaURL MediaURL) ([]byte, error) {
	return json.Marshal(map[string]any{"type": partType, field: mediaURL})
}

type userInputKind uint8

const (
	userInputInvalid userInputKind = iota
	userInputText
	userInputParts
)

// UserInput represents Python's str | list[ContentPart] union.
type UserInput struct {
	kind  userInputKind
	text  string
	parts []ContentPart
}

func NewTextInput(text string) UserInput {
	return UserInput{kind: userInputText, text: text}
}

func NewPartsInput(parts ...ContentPart) UserInput {
	copied := append([]ContentPart(nil), parts...)
	return UserInput{kind: userInputParts, parts: copied}
}

func (u UserInput) Text() (string, bool) {
	return u.text, u.kind == userInputText
}

func (u UserInput) Parts() ([]ContentPart, bool) {
	if u.kind != userInputParts {
		return nil, false
	}
	return append([]ContentPart(nil), u.parts...), true
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

var (
	_ ContentPart = (*TextPart)(nil)
	_ ContentPart = (*ThinkPart)(nil)
	_ ContentPart = (*ImageURLPart)(nil)
	_ ContentPart = (*AudioURLPart)(nil)
	_ ContentPart = (*VideoURLPart)(nil)
	_ Mergeable   = (*TextPart)(nil)
	_ Mergeable   = (*ThinkPart)(nil)
	_ Mergeable   = (*ImageURLPart)(nil)
	_ Mergeable   = (*AudioURLPart)(nil)
	_ Mergeable   = (*VideoURLPart)(nil)
)
