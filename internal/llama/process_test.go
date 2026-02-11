package llama

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

func TestNewProcessSetsPath(t *testing.T) {
	p := NewProcess("/usr/local/bin/llama-server")

	if p.path != "/usr/local/bin/llama-server" {
		t.Errorf("path = %q, want %q", p.path, "/usr/local/bin/llama-server")
	}
}

func TestSetLogWriter(t *testing.T) {
	p := NewProcess("llama-server")
	var buf bytes.Buffer

	p.SetLogWriter(&buf)

	if p.logWriter != &buf {
		t.Error("logWriter was not set")
	}
}

func TestIsRunningReturnsFalseWhenNotStarted(t *testing.T) {
	p := NewProcess("llama-server")

	if p.IsRunning() {
		t.Error("IsRunning() = true, want false for unstarted process")
	}
}

func TestStopReturnsNilWhenNotStarted(t *testing.T) {
	p := NewProcess("llama-server")

	err := p.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() = %v, want nil for unstarted process", err)
	}
}

func TestConcurrentIsRunningCallsNoRace(t *testing.T) {
	p := NewProcess("llama-server")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.IsRunning()
		}()
	}
	wg.Wait()
}

func TestStartReturnsErrorWhenAlreadyRunning(t *testing.T) {
	p := NewProcess("/bin/sleep")

	// Start a real process
	err := p.Start(context.Background(), []string{"60"})
	if err != nil {
		t.Fatalf("first Start() failed: %v", err)
	}
	defer p.Stop(context.Background())

	// Second start should fail
	err = p.Start(context.Background(), []string{"60"})
	if err == nil {
		t.Fatal("second Start() should return error, got nil")
	}
	if err.Error() != "process already running" {
		t.Errorf("got error %q, want %q", err.Error(), "process already running")
	}
}

func TestConcurrentSetLogWriterAndIsRunningNoRace(t *testing.T) {
	p := NewProcess("llama-server")
	var buf bytes.Buffer

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			p.SetLogWriter(&buf)
		}()
		go func() {
			defer wg.Done()
			p.IsRunning()
		}()
	}
	wg.Wait()
}
