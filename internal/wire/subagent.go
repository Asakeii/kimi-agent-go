package wire

import (
	"encoding/json"
	"fmt"
)

type SubagentEvent struct {
	eventMarker
	ParentToolCallID *string `json:"parent_tool_call_id"`
	AgentID          *string `json:"agent_id"`
	SubagentType     *string `json:"subagent_type"`
	Event            Event   `json:"event"`
}

func (*SubagentEvent) wireType() string { return "SubagentEvent" }

func (e SubagentEvent) MarshalJSON() ([]byte, error) {
	if e.Event == nil {
		return nil, fmt.Errorf("wire: subagent event cannot be nil")
	}
	envelope, err := Encode(e.Event)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ParentToolCallID *string  `json:"parent_tool_call_id"`
		AgentID          *string  `json:"agent_id"`
		SubagentType     *string  `json:"subagent_type"`
		Event            Envelope `json:"event"`
	}{e.ParentToolCallID, e.AgentID, e.SubagentType, envelope})
}

func (e *SubagentEvent) UnmarshalJSON(data []byte) error {
	var raw struct {
		ParentToolCallID *string  `json:"parent_tool_call_id"`
		TaskToolCallID   *string  `json:"task_tool_call_id"`
		AgentID          *string  `json:"agent_id"`
		SubagentType     *string  `json:"subagent_type"`
		Event            Envelope `json:"event"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.ParentToolCallID == nil {
		raw.ParentToolCallID = raw.TaskToolCallID
	}
	message, err := Decode(raw.Event)
	if err != nil {
		return err
	}
	event, ok := message.(Event)
	if !ok {
		return fmt.Errorf("wire: SubagentEvent payload must be an Event")
	}
	e.ParentToolCallID = raw.ParentToolCallID
	e.AgentID = raw.AgentID
	e.SubagentType = raw.SubagentType
	e.Event = event
	return nil
}
