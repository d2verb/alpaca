package daemon

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/protocol"
)

func TestClassifyLoadError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantCode     string
		wantContains string
	}{
		{
			name:         "preset not found",
			err:          fmt.Errorf("load preset: %w", &preset.NotFoundError{Name: "test"}),
			wantCode:     protocol.ErrCodePresetNotFound,
			wantContains: "not found",
		},
		{
			name:         "model not found",
			err:          fmt.Errorf("resolve model: %w", &metadata.NotFoundError{Repo: "unknown", Quant: "Q4_K_M"}),
			wantCode:     protocol.ErrCodeModelNotFound,
			wantContains: "not found in metadata",
		},
		{
			name:         "server start failed",
			err:          &llama.ProcessError{Op: llama.ProcessOpStart, Err: fmt.Errorf("command not found")},
			wantCode:     protocol.ErrCodeServerFailed,
			wantContains: "start llama-server",
		},
		{
			name:         "server health check failed",
			err:          &llama.ProcessError{Op: llama.ProcessOpWait, Err: fmt.Errorf("timeout")},
			wantCode:     protocol.ErrCodeServerFailed,
			wantContains: "wait llama-server",
		},
		{
			name:         "unknown error",
			err:          fmt.Errorf("some other error"),
			wantCode:     "",
			wantContains: "some other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			code, msg := classifyLoadError(tt.err)

			// Assert
			if code != tt.wantCode {
				t.Errorf("classifyLoadError() code = %q, want %q", code, tt.wantCode)
			}
			if msg != tt.err.Error() {
				t.Errorf("classifyLoadError() msg = %q, want %q", msg, tt.err.Error())
			}
			if tt.wantContains != "" && !strings.Contains(msg, tt.wantContains) {
				t.Errorf("message %q does not contain %q", msg, tt.wantContains)
			}
		})
	}
}

func TestHandleRequest_Status(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{Command: protocol.CmdStatus}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_Load(t *testing.T) {
	// Arrange
	testPreset := &preset.Preset{
		Name:  "test-preset",
		Model: "f:/path/to/model.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"test-preset": testPreset,
		},
	}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	// Mock dependencies
	mockProc := &mockProcess{}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "p:test-preset",
		},
	}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_Unload(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{Command: protocol.CmdUnload}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_ListPresets(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		names: []string{"test"},
	}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{Command: protocol.CmdListPresets}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_ListModels(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{Command: protocol.CmdListModels}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleRequest_UnknownCommand(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{Command: "unknown_command"}

	// Act
	resp := server.handleRequest(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "unknown command" {
		t.Errorf("Error = %q, want %q", resp.Error, "unknown command")
	}
}
