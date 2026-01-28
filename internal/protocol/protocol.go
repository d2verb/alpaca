// Package protocol defines the JSON protocol for daemon communication.
package protocol

// Request represents a command request to the daemon.
type Request struct {
	Command string         `json:"command"`
	Args    map[string]any `json:"args,omitempty"`
}

// Response represents a response from the daemon.
type Response struct {
	Status string         `json:"status"` // "ok" or "error"
	Data   map[string]any `json:"data,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// Command names
const (
	CmdStatus      = "status"
	CmdLoad        = "load"
	CmdUnload      = "unload"
	CmdListPresets = "list_presets"
	CmdListModels  = "list_models"
)

// Status values
const (
	StatusOK    = "ok"
	StatusError = "error"
)

// NewRequest creates a new request with the given command and args.
func NewRequest(command string, args map[string]any) *Request {
	return &Request{
		Command: command,
		Args:    args,
	}
}

// NewOKResponse creates a successful response with data.
func NewOKResponse(data map[string]any) *Response {
	return &Response{
		Status: StatusOK,
		Data:   data,
	}
}

// NewErrorResponse creates an error response.
func NewErrorResponse(err string) *Response {
	return &Response{
		Status: StatusError,
		Error:  err,
	}
}
