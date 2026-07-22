package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

type ApprovalKind string

const (
	ApprovalApprove           ApprovalKind = "approve"
	ApprovalApproveForSession ApprovalKind = "approve_for_session"
	ApprovalReject            ApprovalKind = "reject"
)

type ApprovalResponse struct {
	eventMarker
	RequestID string       `json:"request_id"`
	Response  ApprovalKind `json:"response"`
	Feedback  string       `json:"feedback"`
}

func (*ApprovalResponse) wireType() string { return "ApprovalResponse" }

func (r *ApprovalResponse) validate() error {
	return validateApprovalKind(r.Response)
}

type approvalResolution struct {
	response ApprovalKind
	feedback string
}

type ApprovalRequest struct {
	requestMarker
	ID                string         `json:"id"`
	ToolCallID        string         `json:"tool_call_id"`
	Sender            string         `json:"sender"`
	Action            string         `json:"action"`
	Description       string         `json:"description"`
	SourceKind        *string        `json:"source_kind"`
	SourceID          *string        `json:"source_id"`
	AgentID           *string        `json:"agent_id"`
	SubagentType      *string        `json:"subagent_type"`
	SourceDescription *string        `json:"source_description"`
	Display           []DisplayBlock `json:"display"`
	result            future[approvalResolution]
}

func (*ApprovalRequest) wireType() string    { return "ApprovalRequest" }
func (r *ApprovalRequest) RequestID() string { return r.ID }
func (r *ApprovalRequest) Resolved() bool    { return r.result.isResolved() }

func (r *ApprovalRequest) Wait(ctx context.Context) (ApprovalKind, error) {
	result, err := r.result.wait(ctx)
	return result.response, err
}

func (r *ApprovalRequest) Resolve(response ApprovalKind, feedback string) bool {
	if validateApprovalKind(response) != nil {
		return false
	}
	return r.result.resolve(approvalResolution{response: response, feedback: feedback}, nil)
}

func (r *ApprovalRequest) Feedback() string {
	result, resolved := r.result.resolvedValue()
	if !resolved {
		return ""
	}
	return result.feedback
}

func (r *ApprovalRequest) MarshalJSON() ([]byte, error) {
	display := r.Display
	if display == nil {
		display = []DisplayBlock{}
	}
	return json.Marshal(struct {
		ID                string         `json:"id"`
		ToolCallID        string         `json:"tool_call_id"`
		Sender            string         `json:"sender"`
		Action            string         `json:"action"`
		Description       string         `json:"description"`
		SourceKind        *string        `json:"source_kind"`
		SourceID          *string        `json:"source_id"`
		AgentID           *string        `json:"agent_id"`
		SubagentType      *string        `json:"subagent_type"`
		SourceDescription *string        `json:"source_description"`
		Display           []DisplayBlock `json:"display"`
	}{
		r.ID, r.ToolCallID, r.Sender, r.Action, r.Description,
		r.SourceKind, r.SourceID, r.AgentID, r.SubagentType, r.SourceDescription, display,
	})
}

func (r *ApprovalRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID                string            `json:"id"`
		ToolCallID        string            `json:"tool_call_id"`
		Sender            string            `json:"sender"`
		Action            string            `json:"action"`
		Description       string            `json:"description"`
		SourceKind        *string           `json:"source_kind"`
		SourceID          *string           `json:"source_id"`
		AgentID           *string           `json:"agent_id"`
		SubagentType      *string           `json:"subagent_type"`
		SourceDescription *string           `json:"source_description"`
		Display           []json.RawMessage `json:"display"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.ID = raw.ID
	r.ToolCallID = raw.ToolCallID
	r.Sender = raw.Sender
	r.Action = raw.Action
	r.Description = raw.Description
	r.SourceKind = raw.SourceKind
	r.SourceID = raw.SourceID
	r.AgentID = raw.AgentID
	r.SubagentType = raw.SubagentType
	r.SourceDescription = raw.SourceDescription
	r.Display = make([]DisplayBlock, 0, len(raw.Display))
	for i, encoded := range raw.Display {
		block, err := decodeDisplayBlock(encoded)
		if err != nil {
			return fmt.Errorf("wire: decode approval display block %d: %w", i, err)
		}
		r.Display = append(r.Display, block)
	}
	return nil
}

type QuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type QuestionItem struct {
	Question         string           `json:"question"`
	Header           string           `json:"header"`
	Options          []QuestionOption `json:"options"`
	MultiSelect      bool             `json:"multi_select"`
	Body             string           `json:"body"`
	OtherLabel       string           `json:"other_label"`
	OtherDescription string           `json:"other_description"`
}

type QuestionResponse struct {
	RequestID string            `json:"request_id"`
	Answers   map[string]string `json:"answers"`
}

var ErrQuestionNotSupported = errors.New("wire: interactive questions are not supported")

type QuestionRequest struct {
	requestMarker
	ID         string         `json:"id"`
	ToolCallID string         `json:"tool_call_id"`
	Questions  []QuestionItem `json:"questions"`
	result     future[map[string]string]
}

func (*QuestionRequest) wireType() string    { return "QuestionRequest" }
func (r *QuestionRequest) RequestID() string { return r.ID }
func (r *QuestionRequest) Resolved() bool    { return r.result.isResolved() }

func (r *QuestionRequest) Wait(ctx context.Context) (map[string]string, error) {
	answers, err := r.result.wait(ctx)
	return cloneStringMap(answers), err
}

func (r *QuestionRequest) Resolve(answers map[string]string) bool {
	return r.result.resolve(cloneStringMap(answers), nil)
}

func (r *QuestionRequest) Reject(err error) bool {
	if err == nil {
		err = ErrQuestionNotSupported
	}
	return r.result.resolve(nil, err)
}

type ToolCallRequest struct {
	requestMarker
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Arguments *string `json:"arguments"`
	result    future[ToolReturnValue]
}

func NewToolCallRequest(call *ToolCall) *ToolCallRequest {
	if call == nil {
		return nil
	}
	return &ToolCallRequest{
		ID: call.ID, Name: call.Function.Name, Arguments: cloneString(call.Function.Arguments),
	}
}

func (*ToolCallRequest) wireType() string    { return "ToolCallRequest" }
func (r *ToolCallRequest) RequestID() string { return r.ID }
func (r *ToolCallRequest) Resolved() bool    { return r.result.isResolved() }

func (r *ToolCallRequest) Wait(ctx context.Context) (ToolReturnValue, error) {
	return r.result.wait(ctx)
}

func (r *ToolCallRequest) Resolve(result ToolReturnValue) bool {
	return r.result.resolve(result, nil)
}

type HookAction string

const (
	HookAllow HookAction = "allow"
	HookBlock HookAction = "block"
)

type HookResponse struct {
	RequestID string     `json:"request_id"`
	Action    HookAction `json:"action"`
	Reason    string     `json:"reason"`
}

type HookRequest struct {
	requestMarker
	ID             string         `json:"id"`
	SubscriptionID string         `json:"subscription_id"`
	Event          string         `json:"event"`
	Target         string         `json:"target"`
	InputData      map[string]any `json:"input_data"`
	result         future[HookResponse]
}

func (*HookRequest) wireType() string    { return "HookRequest" }
func (r *HookRequest) RequestID() string { return r.ID }
func (r *HookRequest) Resolved() bool    { return r.result.isResolved() }

func (r *HookRequest) Wait(ctx context.Context) (HookResponse, error) {
	return r.result.wait(ctx)
}

func (r *HookRequest) Resolve(action HookAction, reason string) bool {
	if action != HookAllow && action != HookBlock {
		return false
	}
	return r.result.resolve(HookResponse{RequestID: r.ID, Action: action, Reason: reason}, nil)
}

func (r *HookRequest) MarshalJSON() ([]byte, error) {
	inputData := r.InputData
	if inputData == nil {
		inputData = map[string]any{}
	}
	return json.Marshal(struct {
		ID             string         `json:"id"`
		SubscriptionID string         `json:"subscription_id"`
		Event          string         `json:"event"`
		Target         string         `json:"target"`
		InputData      map[string]any `json:"input_data"`
	}{r.ID, r.SubscriptionID, r.Event, r.Target, inputData})
}

func validateApprovalKind(kind ApprovalKind) error {
	switch kind {
	case ApprovalApprove, ApprovalApproveForSession, ApprovalReject:
		return nil
	default:
		return fmt.Errorf("wire: invalid approval response %q", kind)
	}
}

func cloneStringMap(value map[string]string) map[string]string {
	if value == nil {
		return nil
	}
	cloned := make(map[string]string, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

var (
	_ Event   = (*ApprovalResponse)(nil)
	_ Request = (*ApprovalRequest)(nil)
	_ Request = (*QuestionRequest)(nil)
	_ Request = (*ToolCallRequest)(nil)
	_ Request = (*HookRequest)(nil)
)
