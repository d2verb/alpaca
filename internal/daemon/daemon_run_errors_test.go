package daemon

import (
	"context"
	"fmt"
	"testing"

	"github.com/d2verb/alpaca/internal/preset"
)

func TestDaemonRun_PresetNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:nonexistent")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q on error", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil on error")
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when preset not found")
	}
}

func TestDaemonRun_ModelNotFound(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		exists: false,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "h:unknown/repo:Q4_K_M")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q on error", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil on error")
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when model not found")
	}
}

func TestDaemonRun_ProcessStartFailure(t *testing.T) {
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
		startErr: fmt.Errorf("failed to start process"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after start failure", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after start failure")
	}
	if !mockProc.startCalled {
		t.Error("Process.Start() should be called")
	}
}

func TestDaemonRun_HealthCheckTimeout(t *testing.T) {
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
	d.waitForReady = mockHealthChecker(fmt.Errorf("health check timeout"))

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after health check failure", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after health check failure")
	}
	if !mockProc.startCalled {
		t.Error("Process.Start() should be called")
	}
	if !mockProc.stopCalled {
		t.Error("Process.Stop() should be called to cleanup after health check failure")
	}
}

func TestDaemonRun_InvalidIdentifier(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "invalid:format:too:many:colons")

	// Assert
	if err == nil {
		t.Fatal("expected error for invalid identifier, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called for invalid identifier")
	}
}

func TestDaemonRun_PresetWithHFModel(t *testing.T) {
	// Arrange - Preset that references a HuggingFace model
	testPreset := &preset.Preset{
		Name:  "codellama-preset",
		Model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M", // HF format in preset
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"codellama-preset": testPreset,
		},
	}
	models := &stubModelManager{
		filePath: "/models/codellama.gguf",
		exists:   true,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:codellama-preset")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	// Verify that the preset's HF model was resolved to a file path
	loadedPreset := d.CurrentPreset()
	if loadedPreset == nil {
		t.Fatal("CurrentPreset() should not be nil")
	}
	// The model field should be resolved from h: to f:
	if loadedPreset.Model != "f:/models/codellama.gguf" {
		t.Errorf("Preset.Model = %q, want %q", loadedPreset.Model, "f:/models/codellama.gguf")
	}
}

func TestDaemonRun_PresetWithHFModelNotFound(t *testing.T) {
	// Arrange - Preset with HF model that doesn't exist
	testPreset := &preset.Preset{
		Name:  "missing-preset",
		Model: "h:unknown/repo:Q4_K_M",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"missing-preset": testPreset,
		},
	}
	models := &stubModelManager{
		exists: false,
	}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	err := d.Run(context.Background(), "p:missing-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error when preset's HF model not found, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q on error", d.State(), StateIdle)
	}
	if mockProc.startCalled {
		t.Error("Process.Start() should not be called when model resolution fails")
	}
}
