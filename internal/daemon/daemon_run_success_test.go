package daemon

import (
	"context"
	"testing"

	"github.com/d2verb/alpaca/internal/preset"
)

func TestDaemonRun_PresetNameSuccess(t *testing.T) {
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
	d.waitForReady = mockHealthChecker(nil) // Success

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}
	if d.CurrentPreset() != testPreset {
		t.Error("CurrentPreset() should return loaded preset")
	}
	if !mockProc.startCalled {
		t.Error("Process.Start() should be called")
	}
	if mockProc.stopCalled {
		t.Error("Process.Stop() should not be called on success")
	}
}

func TestDaemonRun_FilePathSuccess(t *testing.T) {
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
	err := d.Run(context.Background(), "f:/path/to/custom.gguf")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	p := d.CurrentPreset()
	if p == nil {
		t.Fatal("CurrentPreset() should not be nil")
	}
	if p.Model != "f:/path/to/custom.gguf" {
		t.Errorf("Preset.Model = %q, want %q", p.Model, "f:/path/to/custom.gguf")
	}
	// Host, Port use preset defaults via GetXxx() methods
	if p.GetHost() != preset.DefaultHost {
		t.Errorf("GetHost() = %q, want %q", p.GetHost(), preset.DefaultHost)
	}
	if p.GetPort() != preset.DefaultPort {
		t.Errorf("GetPort() = %d, want %d", p.GetPort(), preset.DefaultPort)
	}
}

func TestDaemonRun_HuggingFaceSuccess(t *testing.T) {
	// Arrange
	presets := &stubPresetLoader{}
	models := &stubModelManager{
		filePath: "/models/codellama-7b.Q4_K_M.gguf",
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
	err := d.Run(context.Background(), "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}

	preset := d.CurrentPreset()
	if preset == nil {
		t.Fatal("CurrentPreset() should not be nil")
	}
	if preset.Name != "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M" {
		t.Errorf("Preset.Name = %q, want %q", preset.Name, "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
	}
	if preset.Model != "f:/models/codellama-7b.Q4_K_M.gguf" {
		t.Errorf("Preset.Model = %q, want %q", preset.Model, "f:/models/codellama-7b.Q4_K_M.gguf")
	}
}
