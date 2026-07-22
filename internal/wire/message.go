package wire

// Message is any value that may travel through a Wire.
// The unexported marker seals the protocol to this package.
type Message interface {
	wireType() string
	isMessage()
}

// Event is a one-way notification from the agent runtime.
type Event interface {
	Message
	isEvent()
}

// Request expects a response correlated by RequestID.
type Request interface {
	Message
	RequestID() string
	isRequest()
}

// ContentPart is model-facing text, reasoning, image, audio, or video content.
type ContentPart interface {
	Event
	isContentPart()
}

// Mergeable is an orthogonal capability used by streamed message fragments.
type Mergeable interface {
	Message
	Clone() Mergeable
	MergeInPlace(next Mergeable) bool
}

type eventMarker struct{}

func (eventMarker) isMessage() {}
func (eventMarker) isEvent()   {}

type requestMarker struct{}

func (requestMarker) isMessage() {}
func (requestMarker) isRequest() {}

type contentPartMarker struct{ eventMarker }

func (contentPartMarker) wireType() string { return "ContentPart" }
func (contentPartMarker) isContentPart()   {}
