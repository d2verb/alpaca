package protocol

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    map[string]any
	}{
		{
			name:    "status command without args",
			command: CmdStatus,
			args:    nil,
		},
		{
			name:    "load command with identifier arg",
			command: CmdLoad,
			args:    map[string]any{"identifier": "codellama-7b"},
		},
		{
			name:    "unload command without args",
			command: CmdUnload,
			args:    nil,
		},
		{
			name:    "list_presets command",
			command: CmdListPresets,
			args:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewRequest(tt.command, tt.args)

			if req.Command != tt.command {
				t.Errorf("Command = %q, want %q", req.Command, tt.command)
			}
			if tt.args == nil && req.Args != nil {
				t.Errorf("Args = %v, want nil", req.Args)
			}
			if tt.args != nil {
				for k, v := range tt.args {
					if req.Args[k] != v {
						t.Errorf("Args[%q] = %v, want %v", k, req.Args[k], v)
					}
				}
			}
		})
	}
}

func TestNewOKResponse(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{
			name: "with nil data",
			data: nil,
		},
		{
			name: "with state data",
			data: map[string]any{"state": "running"},
		},
		{
			name: "with multiple fields",
			data: map[string]any{
				"state":    "running",
				"preset":   "codellama-7b",
				"endpoint": "http://localhost:8080",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewOKResponse(tt.data)

			if resp.Status != StatusOK {
				t.Errorf("Status = %q, want %q", resp.Status, StatusOK)
			}
			if resp.Error != "" {
				t.Errorf("Error = %q, want empty", resp.Error)
			}
			if tt.data == nil && resp.Data != nil {
				t.Errorf("Data = %v, want nil", resp.Data)
			}
			if tt.data != nil {
				for k, v := range tt.data {
					if resp.Data[k] != v {
						t.Errorf("Data[%q] = %v, want %v", k, resp.Data[k], v)
					}
				}
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		err  string
	}{
		{"empty error", ""},
		{"simple error", "connection refused"},
		{"detailed error", "preset 'foo' not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewErrorResponse(tt.err)

			if resp.Status != StatusError {
				t.Errorf("Status = %q, want %q", resp.Status, StatusError)
			}
			if resp.Error != tt.err {
				t.Errorf("Error = %q, want %q", resp.Error, tt.err)
			}
			if resp.Data != nil {
				t.Errorf("Data = %v, want nil", resp.Data)
			}
		})
	}
}

func TestRequest_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		req  *Request
	}{
		{
			name: "status request",
			req:  NewRequest(CmdStatus, nil),
		},
		{
			name: "load request with args",
			req:  NewRequest(CmdLoad, map[string]any{"identifier": "test"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded Request
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.Command != tt.req.Command {
				t.Errorf("Command = %q, want %q", decoded.Command, tt.req.Command)
			}
		})
	}
}

func TestResponse_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		resp *Response
	}{
		{
			name: "ok response",
			resp: NewOKResponse(map[string]any{"state": "idle"}),
		},
		{
			name: "error response",
			resp: NewErrorResponse("something went wrong"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("Marshal error: %v", err)
			}

			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Unmarshal error: %v", err)
			}

			if decoded.Status != tt.resp.Status {
				t.Errorf("Status = %q, want %q", decoded.Status, tt.resp.Status)
			}
			if decoded.Error != tt.resp.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.resp.Error)
			}
		})
	}
}
