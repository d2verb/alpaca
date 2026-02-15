package daemon

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/d2verb/alpaca/internal/preset"
)

func TestDaemonRun_StopsExistingModel(t *testing.T) {
	// Arrange
	firstPreset := &preset.Preset{
		Name:  "first-preset",
		Model: "f:/path/to/first.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}
	secondPreset := &preset.Preset{
		Name:  "second-preset",
		Model: "f:/path/to/second.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"first-preset":  firstPreset,
			"second-preset": secondPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	firstMockProc := &mockProcess{}
	secondMockProc := &mockProcess{}
	callCount := 0
	d.newProcess = func(path string) llamaProcess {
		callCount++
		if callCount == 1 {
			return firstMockProc
		}
		return secondMockProc
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	// Load first model
	err := d.Run(context.Background(), "p:first-preset")
	if err != nil {
		t.Fatalf("first Run() failed: %v", err)
	}

	// Load second model (should stop first)
	err = d.Run(context.Background(), "p:second-preset")

	// Assert
	if err != nil {
		t.Fatalf("second Run() failed: %v", err)
	}
	if !firstMockProc.stopCalled {
		t.Error("first process should be stopped when loading second model")
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}
	if d.CurrentPreset() != secondPreset {
		t.Error("CurrentPreset() should be second preset")
	}
}

func TestDaemonRun_FailsToStopExistingModel(t *testing.T) {
	// Arrange
	firstPreset := &preset.Preset{
		Name:  "first-preset",
		Model: "f:/path/to/first.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}
	secondPreset := &preset.Preset{
		Name:  "second-preset",
		Model: "f:/path/to/second.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}

	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"first-preset":  firstPreset,
			"second-preset": secondPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Mock dependencies
	firstMockProc := &mockProcess{
		stopErr: fmt.Errorf("failed to stop"),
	}
	callCount := 0
	d.newProcess = func(path string) llamaProcess {
		callCount++
		if callCount == 1 {
			return firstMockProc
		}
		return &mockProcess{}
	}
	d.waitForReady = mockHealthChecker(nil)

	// Act
	// Load first model
	err := d.Run(context.Background(), "p:first-preset")
	if err != nil {
		t.Fatalf("first Run() failed: %v", err)
	}

	// Try to load second model (should fail to stop first)
	err = d.Run(context.Background(), "p:second-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error when stopping existing model fails, got nil")
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q (should remain in previous state)", d.State(), StateRunning)
	}
	if d.CurrentPreset() != firstPreset {
		t.Error("CurrentPreset() should still be first preset after failed stop")
	}
}

func TestDaemonRun_ContextCancelledDuringHealthCheck(t *testing.T) {
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
	d.waitForReady = mockHealthChecker(context.Canceled)

	// Act
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	err := d.Run(ctx, "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error when context is cancelled, got nil")
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q after context cancellation", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after context cancellation")
	}
	if mockProc.stopCalled {
		t.Log("Process.Stop() was called for cleanup (acceptable)")
	}
}

func TestDaemonRun_ProcessDiesDuringStartup(t *testing.T) {
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

	doneCh := make(chan struct{})
	close(doneCh) // simulate immediate process death
	mockProc := &mockProcess{
		doneCh:    doneCh,
		exitError: fmt.Errorf("exit status 1"),
	}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = func(ctx context.Context, endpoint string) error {
		<-ctx.Done()
		return ctx.Err()
	}

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "llama-server exited unexpectedly") {
		t.Errorf("error should contain 'llama-server exited unexpectedly', got: %s", err)
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil after process death")
	}
}

func TestDaemonRun_StartupTimeout(t *testing.T) {
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
	d.startupTimeout = 50 * time.Millisecond // short timeout for test

	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = func(ctx context.Context, endpoint string) error {
		<-ctx.Done()
		return ctx.Err()
	}

	// Act
	err := d.Run(context.Background(), "p:test-preset")

	// Assert
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "did not become ready") {
		t.Errorf("error should contain user-friendly timeout message, got: %s", err)
	}
	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
}

func TestDaemonKill_DuringStartup(t *testing.T) {
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

	healthCheckStarted := make(chan struct{})
	mockProc := &mockProcess{}
	d.newProcess = func(path string) llamaProcess {
		return mockProc
	}
	d.waitForReady = func(ctx context.Context, endpoint string) error {
		close(healthCheckStarted)
		<-ctx.Done()
		return ctx.Err()
	}

	// Act — start Run() in background, then Kill()
	runDone := make(chan error, 1)
	go func() {
		runDone <- d.Run(context.Background(), "p:test-preset")
	}()

	<-healthCheckStarted
	err := d.Kill(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("Kill() error: %v", err)
	}

	select {
	case runErr := <-runDone:
		if runErr == nil {
			t.Fatal("Run() should return error when killed during startup")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after Kill()")
	}

	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
}

func TestDaemonRun_CancelsPreviousStartup(t *testing.T) {
	// Arrange
	firstPreset := &preset.Preset{
		Name:  "first-preset",
		Model: "f:/path/to/first.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}
	secondPreset := &preset.Preset{
		Name:  "second-preset",
		Model: "f:/path/to/second.gguf",
		Host:  "127.0.0.1",
		Port:  8080,
	}
	presets := &stubPresetLoader{
		presets: map[string]*preset.Preset{
			"first-preset":  firstPreset,
			"second-preset": secondPreset,
		},
	}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	healthCheckStarted := make(chan struct{})
	firstCall := true
	d.newProcess = func(path string) llamaProcess {
		return &mockProcess{}
	}
	d.waitForReady = func(ctx context.Context, endpoint string) error {
		if firstCall {
			firstCall = false
			close(healthCheckStarted)
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	}

	// Act — start first Run() in background
	firstRunDone := make(chan error, 1)
	go func() {
		firstRunDone <- d.Run(context.Background(), "p:first-preset")
	}()

	<-healthCheckStarted
	err := d.Run(context.Background(), "p:second-preset")

	// Assert
	if err != nil {
		t.Fatalf("second Run() error: %v", err)
	}
	if d.State() != StateRunning {
		t.Errorf("State() = %q, want %q", d.State(), StateRunning)
	}
	if d.CurrentPreset().Name != "second-preset" {
		t.Errorf("CurrentPreset().Name = %q, want %q", d.CurrentPreset().Name, "second-preset")
	}

	select {
	case firstErr := <-firstRunDone:
		if firstErr == nil {
			t.Fatal("first Run() should return error when cancelled")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first Run() did not return after cancellation")
	}
}
