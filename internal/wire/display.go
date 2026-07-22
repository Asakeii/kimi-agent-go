package wire

import (
	"encoding/json"
	"fmt"
)

// DisplayBlock is UI-facing tool output. It is deliberately separate from model-facing ContentPart.
type DisplayBlock interface {
	displayType() string
	isDisplayBlock()
}

type displayBlockMarker struct{}

func (*displayBlockMarker) isDisplayBlock() {}

type UnknownDisplayBlock struct {
	displayBlockMarker
	Type string `json:"type"`
	Data any    `json:"data"`
}

type BriefDisplayBlock struct {
	displayBlockMarker
	Text string `json:"text"`
}

type DiffDisplayBlock struct {
	displayBlockMarker
	Path      string `json:"path"`
	OldText   string `json:"old_text"`
	NewText   string `json:"new_text"`
	OldStart  int    `json:"old_start"`
	NewStart  int    `json:"new_start"`
	IsSummary bool   `json:"is_summary"`
}

type TodoDisplayItem struct {
	Title  string `json:"title"`
	Status string `json:"status"`
}

type TodoDisplayBlock struct {
	displayBlockMarker
	Items []TodoDisplayItem `json:"items"`
}

type ShellDisplayBlock struct {
	displayBlockMarker
	Language string `json:"language"`
	Command  string `json:"command"`
}

type BackgroundTaskDisplayBlock struct {
	displayBlockMarker
	TaskID      string `json:"task_id"`
	Kind        string `json:"kind"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

func (b *UnknownDisplayBlock) displayType() string      { return b.Type }
func (*BriefDisplayBlock) displayType() string          { return "brief" }
func (*DiffDisplayBlock) displayType() string           { return "diff" }
func (*TodoDisplayBlock) displayType() string           { return "todo" }
func (*ShellDisplayBlock) displayType() string          { return "shell" }
func (*BackgroundTaskDisplayBlock) displayType() string { return "background_task" }

func (b UnknownDisplayBlock) MarshalJSON() ([]byte, error) {
	type payload struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}
	return json.Marshal(payload{Type: b.Type, Data: b.Data})
}

func (b BriefDisplayBlock) MarshalJSON() ([]byte, error) {
	return marshalTypedDisplayBlock("brief", struct {
		Text string `json:"text"`
	}{Text: b.Text})
}

func (b DiffDisplayBlock) MarshalJSON() ([]byte, error) {
	if b.OldStart == 0 {
		b.OldStart = 1
	}
	if b.NewStart == 0 {
		b.NewStart = 1
	}
	return marshalTypedDisplayBlock("diff", struct {
		Path      string `json:"path"`
		OldText   string `json:"old_text"`
		NewText   string `json:"new_text"`
		OldStart  int    `json:"old_start"`
		NewStart  int    `json:"new_start"`
		IsSummary bool   `json:"is_summary"`
	}{b.Path, b.OldText, b.NewText, b.OldStart, b.NewStart, b.IsSummary})
}

func (b TodoDisplayBlock) MarshalJSON() ([]byte, error) {
	if b.Items == nil {
		b.Items = []TodoDisplayItem{}
	}
	return marshalTypedDisplayBlock("todo", struct {
		Items []TodoDisplayItem `json:"items"`
	}{Items: b.Items})
}

func (b ShellDisplayBlock) MarshalJSON() ([]byte, error) {
	return marshalTypedDisplayBlock("shell", struct {
		Language string `json:"language"`
		Command  string `json:"command"`
	}{b.Language, b.Command})
}

func (b BackgroundTaskDisplayBlock) MarshalJSON() ([]byte, error) {
	return marshalTypedDisplayBlock("background_task", struct {
		TaskID      string `json:"task_id"`
		Kind        string `json:"kind"`
		Status      string `json:"status"`
		Description string `json:"description"`
	}{b.TaskID, b.Kind, b.Status, b.Description})
}

func marshalTypedDisplayBlock(blockType string, value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		return nil, err
	}
	fields["type"] = blockType
	return json.Marshal(fields)
}

func decodeDisplayBlock(data json.RawMessage) (DisplayBlock, error) {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("wire: decode display block header: %w", err)
	}
	if header.Type == "" {
		return nil, fmt.Errorf("wire: display block type is empty")
	}

	var block DisplayBlock
	switch header.Type {
	case "brief":
		block = &BriefDisplayBlock{}
	case "diff":
		block = &DiffDisplayBlock{OldStart: 1, NewStart: 1}
	case "todo":
		block = &TodoDisplayBlock{}
	case "shell":
		block = &ShellDisplayBlock{}
	case "background_task":
		block = &BackgroundTaskDisplayBlock{}
	default:
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		delete(raw, "type")
		return &UnknownDisplayBlock{Type: header.Type, Data: raw}, nil
	}
	if err := json.Unmarshal(data, block); err != nil {
		return nil, fmt.Errorf("wire: decode %s display block: %w", header.Type, err)
	}
	return block, nil
}
