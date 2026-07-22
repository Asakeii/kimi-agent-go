package wire

import (
	"encoding/json"
	"fmt"
)

type TurnBegin struct {
	eventMarker
	UserInput UserInput `json:"user_input"`
}

type SteerInput struct {
	eventMarker
	UserInput UserInput `json:"user_input"`
}

type TurnEnd struct{ eventMarker }

type StepBegin struct {
	eventMarker
	N int `json:"n"`
}

type StepInterrupted struct{ eventMarker }

type StepRetry struct {
	eventMarker
	N           int     `json:"n"`
	NextAttempt int     `json:"next_attempt"`
	MaxAttempts int     `json:"max_attempts"`
	WaitSeconds float64 `json:"wait_s"`
	ErrorType   string  `json:"error_type"`
	StatusCode  *int    `json:"status_code"`
}

type CompactionBegin struct{ eventMarker }
type CompactionEnd struct{ eventMarker }
type MCPLoadingBegin struct{ eventMarker }
type MCPLoadingEnd struct{ eventMarker }

type HookTriggered struct {
	eventMarker
	Event     string `json:"event"`
	Target    string `json:"target"`
	HookCount int    `json:"hook_count"`
}

type HookResolved struct {
	eventMarker
	Event      string `json:"event"`
	Target     string `json:"target"`
	Action     string `json:"action"`
	Reason     string `json:"reason"`
	DurationMS int    `json:"duration_ms"`
}

type MCPServerSnapshot struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Tools  []string `json:"tools"`
}

type MCPStatusSnapshot struct {
	Loading   bool                `json:"loading"`
	Connected int                 `json:"connected"`
	Total     int                 `json:"total"`
	Tools     int                 `json:"tools"`
	Servers   []MCPServerSnapshot `json:"servers"`
}

type TokenUsage struct {
	InputOther         int `json:"input_other"`
	Output             int `json:"output"`
	InputCacheRead     int `json:"input_cache_read"`
	InputCacheCreation int `json:"input_cache_creation"`
}

func (u TokenUsage) Input() int {
	return u.InputOther + u.InputCacheRead + u.InputCacheCreation
}

func (u TokenUsage) Total() int {
	return u.Input() + u.Output
}

type StatusUpdate struct {
	eventMarker
	ContextUsage     *float64           `json:"context_usage"`
	ContextTokens    *int               `json:"context_tokens"`
	MaxContextTokens *int               `json:"max_context_tokens"`
	TokenUsage       *TokenUsage        `json:"token_usage"`
	MessageID        *string            `json:"message_id"`
	PlanMode         *bool              `json:"plan_mode"`
	MCPStatus        *MCPStatusSnapshot `json:"mcp_status"`
}

type Notification struct {
	eventMarker
	ID         string         `json:"id"`
	Category   string         `json:"category"`
	Type       string         `json:"type"`
	SourceKind string         `json:"source_kind"`
	SourceID   string         `json:"source_id"`
	Title      string         `json:"title"`
	Body       string         `json:"body"`
	Severity   string         `json:"severity"`
	CreatedAt  float64        `json:"created_at"`
	Payload    map[string]any `json:"payload"`
}

type PlanDisplay struct {
	eventMarker
	Content  string `json:"content"`
	FilePath string `json:"file_path"`
}

type BtwBegin struct {
	eventMarker
	ID       string `json:"id"`
	Question string `json:"question"`
}

type BtwEnd struct {
	eventMarker
	ID       string  `json:"id"`
	Response *string `json:"response"`
	Error    *string `json:"error"`
}

func NewTurnBegin(userInput UserInput) *TurnBegin   { return &TurnBegin{UserInput: userInput} }
func NewSteerInput(userInput UserInput) *SteerInput { return &SteerInput{UserInput: userInput} }

func (*TurnBegin) wireType() string       { return "TurnBegin" }
func (*SteerInput) wireType() string      { return "SteerInput" }
func (*TurnEnd) wireType() string         { return "TurnEnd" }
func (*StepBegin) wireType() string       { return "StepBegin" }
func (*StepInterrupted) wireType() string { return "StepInterrupted" }
func (*StepRetry) wireType() string       { return "StepRetry" }
func (*CompactionBegin) wireType() string { return "CompactionBegin" }
func (*CompactionEnd) wireType() string   { return "CompactionEnd" }
func (*HookTriggered) wireType() string   { return "HookTriggered" }
func (*HookResolved) wireType() string    { return "HookResolved" }
func (*MCPLoadingBegin) wireType() string { return "MCPLoadingBegin" }
func (*MCPLoadingEnd) wireType() string   { return "MCPLoadingEnd" }
func (*StatusUpdate) wireType() string    { return "StatusUpdate" }
func (*Notification) wireType() string    { return "Notification" }
func (*PlanDisplay) wireType() string     { return "PlanDisplay" }
func (*BtwBegin) wireType() string        { return "BtwBegin" }
func (*BtwEnd) wireType() string          { return "BtwEnd" }

func (t *TurnBegin) UnmarshalJSON(data []byte) error {
	return unmarshalUserInputEvent(data, &t.UserInput, "TurnBegin")
}

func (s *SteerInput) UnmarshalJSON(data []byte) error {
	return unmarshalUserInputEvent(data, &s.UserInput, "SteerInput")
}

func unmarshalUserInputEvent(data []byte, target *UserInput, name string) error {
	var payload struct {
		UserInput json.RawMessage `json:"user_input"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if len(payload.UserInput) == 0 {
		return fmt.Errorf("wire: %s is missing user_input", name)
	}
	return json.Unmarshal(payload.UserInput, target)
}

func (s MCPServerSnapshot) MarshalJSON() ([]byte, error) {
	type alias MCPServerSnapshot
	if s.Tools == nil {
		s.Tools = []string{}
	}
	return json.Marshal(alias(s))
}

func (s MCPStatusSnapshot) MarshalJSON() ([]byte, error) {
	type alias MCPStatusSnapshot
	if s.Servers == nil {
		s.Servers = []MCPServerSnapshot{}
	}
	return json.Marshal(alias(s))
}

func (n Notification) MarshalJSON() ([]byte, error) {
	type alias Notification
	if n.Payload == nil {
		n.Payload = map[string]any{}
	}
	return json.Marshal(alias(n))
}
