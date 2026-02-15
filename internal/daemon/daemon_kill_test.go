package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/d2verb/alpaca/internal/preset"
)

func TestDaemonKill_WhenRunning(t *testing.T) {
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
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start a model first
	err := d.Run(context.Background(), "p:test-preset")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	err = d.Kill(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mockProc.stopCalled {
		t.Error("Process.Stop() should be called")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after Kill()", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after Kill()")
	}
}

func TestDaemonKill_WhenIdle(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Act
	err := d.Kill(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Kill() on idle daemon should not error: %v", err)
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
}

func TestDaemonKill_StopError(t *testing.T) {
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
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop process"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Start a model first
	err := d.Run(context.Background(), "p:test-preset")
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Act
	err = d.Kill(context.Background())

	// Assert
	if err == nil {
		t.Fatal("expected error from Kill(), got nil")
	}
	if !mockProc.stopCalled {
		t.Error("Process.Stop() should be called even if it errors")
	}
}
