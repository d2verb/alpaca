package daemon

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/protocol"
)

func TestHandleUnload_Success(t *testing.T) {
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

	// Mock dependencies and start a model first
	mockProc := &mockProcess{}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	err := daemon.Run(context.Background(), "p:test-preset")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	resp := server.handleUnload(context.Background())

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
	if daemon.State() != StateIdle {
		t.Errorf("daemon state = %q, want %q after unload", daemon.State(), StateIdle)
	}
}

func TestHandleUnload_WhenIdle(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	daemon := newTestDaemon(presets, models)
	server := NewServer(daemon, "/tmp/test.sock", io.Discard)

	// Act
	resp := server.handleUnload(context.Background())

	// Assert
	if resp.Status != protocol.StatusOK {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusOK)
	}
}

func TestHandleUnload_Error(t *testing.T) {
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

	// Mock process that fails to stop
	mockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop process"),
	}
	daemon.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	daemon.waitForReady = mockHealthChecker(nil)

	err := daemon.Run(context.Background(), "p:test-preset")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	resp := server.handleUnload(context.Background())

	// Assert
	if resp.Status != protocol.StatusError {
		t.Errorf("Status = %q, want %q", resp.Status, protocol.StatusError)
	}
	if resp.Error != "failed to stop process" {
		t.Errorf("Error = %q, want %q", resp.Error, "failed to stop process")
	}
}
