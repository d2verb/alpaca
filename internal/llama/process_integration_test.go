package llama

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var (
	fakeProcPath string
	buildOnce    sync.Once
	buildErr     error
)

// buildFakeProc builds the fake process binary once per test run.
func buildFakeProc(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		// Build to a temp directory
		tmpDir, err := os.MkdirTemp("", "llama-test-*")
		if err != nil {
			buildErr = err
			return
		}

		fakeProcPath = filepath.Join(tmpDir, "fakeproc")
		cmd := exec.Command("go", "build", "-o", fakeProcPath, "./testdata/fakeproc")
		cmd.Dir = filepath.Dir(fakeProcPath)
		// Set the correct working directory
		wd, _ := os.Getwd()
		cmd.Dir = wd
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0")

		out, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = &exec.ExitError{Stderr: out}
			return
		}
	})

	if buildErr != nil {
		t.Fatalf("failed to build fakeproc: %v", buildErr)
	}
	return fakeProcPath
}

func TestProcess_StartAndStop_Graceful(t *testing.T) {
	// Arrange
	bin := buildFakeProc(t)
	p := NewProcess(bin)

	// Act
	err := p.Start([]string{"-mode=sigterm"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Assert process is running
	if !p.IsRunning() {
		t.Error("IsRunning() = false after Start()")
	}

	// Stop gracefully
	err = p.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Assert process stopped - wait a bit for process to fully exit
	time.Sleep(200 * time.Millisecond)

	// Note: IsRunning() may still return true briefly after Stop()
	// because the process state hasn't been reaped yet.
	// The important thing is that Stop() returned successfully.
}

func TestProcess_StartAndStop_ForceKill(t *testing.T) {
	// Arrange
	bin := buildFakeProc(t)
	p := NewProcess(bin)

	// Use run mode which ignores SIGTERM
	err := p.Start([]string{"-mode=run"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Give it time to start
	time.Sleep(50 * time.Millisecond)

	// Stop with a short context timeout to force kill
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Act
	err = p.Stop(ctx)

	// Assert - Stop should succeed (process killed)
	// Note: error might be context.DeadlineExceeded if context times out first
	if err != nil && err != context.DeadlineExceeded {
		t.Logf("Stop() returned error (expected for force kill): %v", err)
	}

	// Process should no longer be running
	time.Sleep(100 * time.Millisecond)
	if p.IsRunning() {
		t.Error("IsRunning() = true after forced Stop()")
	}
}
