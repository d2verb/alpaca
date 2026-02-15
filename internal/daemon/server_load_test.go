package daemon

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/protocol"
)

func TestHandleLoad_Success(t *testing.T) {
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
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
	if resp.Data["endpoint"] != "http://127.0.0.1:8080" {
		t.Errorf("endpoint = %v, want %q", resp.Data["endpoint"], "http://127.0.0.1:8080")
	}
}

func TestHandleLoad_MissingIdentifier(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args:    map[string]any{}, // No identifier
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "identifier required" {
		t.Errorf("Error = %q, want %q", resp.Error, "identifier required")
	}
}

func TestHandleLoad_PresetNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{},
	}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "p:nonexistent",
		},
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.ErrorCode != protocol.ErrCodePresetNotFound {
		t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodePresetNotFound)
	}
}

func TestHandleLoad_ModelNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		err: &metadata.NotFoundError{Repo: "unknown", Quant: "Q4_K_M"},
	}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	req := &protocol.Request{
		Command: protocol.CmdLoad,
		Args: map[string]any{
			"identifier": "h:unknown/repo:Q4_K_M",
		},
	}

	// Act
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.ErrorCode != protocol.ErrCodeModelNotFound {
		t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodeModelNotFound)
	}
}

func TestHandleLoad_ServerStartFailed(t *testing.T) {
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

	// Mock process that fails to start
	mockProc := &mockProcess{
		startErr: fmt.Errorf("command not found"),
	}
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
	resp := server.handleLoad(context.Background(), req)

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.ErrorCode != protocol.ErrCodeServerFailed {
		t.Errorf("ErrorCode = %q, want %q", resp.ErrorCode, protocol.ErrCodeServerFailed)
	}
}
