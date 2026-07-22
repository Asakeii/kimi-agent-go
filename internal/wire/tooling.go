package wire

import (
	"encoding/json"
	"fmt"
)

type ToolFunction struct {
	Name      string  `json:"name"`
	Arguments *string `json:"arguments"`
}

type ToolCall struct {
	eventMarker
	ID       string         `json:"id"`
	Function ToolFunction   `json:"function"`
	Extras   map[string]any `json:"extras"`
}

type ToolCallPart struct {
	eventMarker
	ArgumentsPart *string `json:"arguments_part"`
}

type toolOutputKind uint8

const (
	toolOutputInvalid toolOutputKind = iota
	toolOutputText
	toolOutputParts
)

// ToolOutput represents Python's str | list[ContentPart] tool output union.
type ToolOutput struct {
	kind  toolOutputKind
	text  string
	parts []ContentPart
}

func NewTextToolOutput(text string) ToolOutput {
	return ToolOutput{kind: toolOutputText, text: text}
}

func NewPartsToolOutput(parts ...ContentPart) ToolOutput {
	copied := make([]ContentPart, len(parts))
	copy(copied, parts)
	return ToolOutput{kind: toolOutputParts, parts: copied}
}

func (o ToolOutput) Text() (string, bool) {
	return o.text, o.kind == toolOutputText
}

func (o ToolOutput) Parts() ([]ContentPart, bool) {
	if o.kind != toolOutputParts {
		return nil, false
	}
	copied := make([]ContentPart, len(o.parts))
	copy(copied, o.parts)
	return copied, true
}

func (o ToolOutput) MarshalJSON() ([]byte, error) {
	switch o.kind {
	case toolOutputText:
		return json.Marshal(o.text)
	case toolOutputParts:
		return json.Marshal(o.parts)
	default:
		return nil, fmt.Errorf("wire: tool output is not initialized")
	}
}

func (o *ToolOutput) UnmarshalJSON(data []byte) error {
	var value UserInput
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if text, ok := value.Text(); ok {
		*o = NewTextToolOutput(text)
		return nil
	}
	parts, _ := value.Parts()
	*o = NewPartsToolOutput(parts...)
	return nil
}

type ToolReturnValue struct {
	IsError bool           `json:"is_error"`
	Output  ToolOutput     `json:"output"`
	Message string         `json:"message"`
	Display []DisplayBlock `json:"display"`
	Extras  map[string]any `json:"extras"`
}

func (v ToolReturnValue) Brief() string {
	for _, block := range v.Display {
		if brief, ok := block.(*BriefDisplayBlock); ok {
			return brief.Text
		}
	}
	return ""
}

func (v ToolReturnValue) MarshalJSON() ([]byte, error) {
	type alias ToolReturnValue
	if v.Display == nil {
		v.Display = []DisplayBlock{}
	}
	return json.Marshal(alias(v))
}

func (v *ToolReturnValue) UnmarshalJSON(data []byte) error {
	var raw struct {
		IsError bool              `json:"is_error"`
		Output  json.RawMessage   `json:"output"`
		Message string            `json:"message"`
		Display []json.RawMessage `json:"display"`
		Extras  map[string]any    `json:"extras"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Output) == 0 {
		return fmt.Errorf("wire: tool return value is missing output")
	}
	if err := json.Unmarshal(raw.Output, &v.Output); err != nil {
		return err
	}
	v.IsError = raw.IsError
	v.Message = raw.Message
	v.Extras = raw.Extras
	v.Display = make([]DisplayBlock, 0, len(raw.Display))
	for i, encoded := range raw.Display {
		block, err := decodeDisplayBlock(encoded)
		if err != nil {
			return fmt.Errorf("wire: decode display block %d: %w", i, err)
		}
		v.Display = append(v.Display, block)
	}
	return nil
}

type ToolResult struct {
	eventMarker
	ToolCallID  string          `json:"tool_call_id"`
	ReturnValue ToolReturnValue `json:"return_value"`
}

func (*ToolCall) wireType() string     { return "ToolCall" }
func (*ToolCallPart) wireType() string { return "ToolCallPart" }
func (*ToolResult) wireType() string   { return "ToolResult" }

func (c *ToolCall) Clone() Mergeable {
	cloned := &ToolCall{ID: c.ID, Function: cloneToolFunction(c.Function)}
	cloned.Extras = cloneJSONMap(c.Extras)
	return cloned
}

func (c *ToolCall) MergeInPlace(next Mergeable) bool {
	part, ok := next.(*ToolCallPart)
	if !ok {
		return false
	}
	if c.Function.Arguments == nil {
		c.Function.Arguments = cloneString(part.ArgumentsPart)
	} else if part.ArgumentsPart != nil {
		*c.Function.Arguments += *part.ArgumentsPart
	}
	return true
}

func (p *ToolCallPart) Clone() Mergeable {
	return &ToolCallPart{ArgumentsPart: cloneString(p.ArgumentsPart)}
}

func (p *ToolCallPart) MergeInPlace(next Mergeable) bool {
	other, ok := next.(*ToolCallPart)
	if !ok {
		return false
	}
	if p.ArgumentsPart == nil {
		p.ArgumentsPart = cloneString(other.ArgumentsPart)
	} else if other.ArgumentsPart != nil {
		*p.ArgumentsPart += *other.ArgumentsPart
	}
	return true
}

func (c ToolCall) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string         `json:"type"`
		ID       string         `json:"id"`
		Function ToolFunction   `json:"function"`
		Extras   map[string]any `json:"extras"`
	}{Type: "function", ID: c.ID, Function: c.Function, Extras: c.Extras})
}

func (c *ToolCall) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type     string         `json:"type"`
		ID       string         `json:"id"`
		Function ToolFunction   `json:"function"`
		Extras   map[string]any `json:"extras"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Type != "function" {
		return fmt.Errorf("wire: invalid tool call type %q", raw.Type)
	}
	c.ID, c.Function, c.Extras = raw.ID, raw.Function, raw.Extras
	return nil
}

func cloneToolFunction(function ToolFunction) ToolFunction {
	return ToolFunction{Name: function.Name, Arguments: cloneString(function.Arguments)}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return nil
	}
	return cloned
}

var (
	_ Event     = (*ToolCall)(nil)
	_ Event     = (*ToolCallPart)(nil)
	_ Event     = (*ToolResult)(nil)
	_ Mergeable = (*ToolCall)(nil)
	_ Mergeable = (*ToolCallPart)(nil)
)
