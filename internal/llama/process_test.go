package llama

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"
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

func TestDoneReturnsNilWhenNotStarted(t *testing.T) {
	p := NewProcess("llama-server")

	if p.Done() != nil {
		t.Error("Done() should return nil for unstarted process")
	}
}

func TestDoneClosesOnNormalExit(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 0"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	select {
	case <-p.Done():
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("Done() was not closed after process exited")
	}
}

func TestDoneClosesOnAbnormalExit(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 1"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	select {
	case <-p.Done():
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("Done() was not closed after process exited abnormally")
	}
}

func TestExitErrNilOnNormalExit(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 0"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	<-p.Done()

	if p.ExitErr() != nil {
		t.Errorf("ExitErr() = %v, want nil for normal exit", p.ExitErr())
	}
}

func TestExitErrSetOnAbnormalExit(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 1"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	<-p.Done()

	if p.ExitErr() == nil {
		t.Fatal("ExitErr() = nil, want error for abnormal exit")
	}
}

func TestStopOnAlreadyExitedProcess(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 0"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	<-p.Done()

	err = p.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() = %v, want nil for already exited process", err)
	}
}

func TestIsRunningTrueWhileProcessRuns(t *testing.T) {
	p := NewProcess("/bin/sleep")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"60"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	defer p.Stop(context.Background())

	if !p.IsRunning() {
		t.Error("IsRunning() = false, want true for running process")
	}
}

func TestIsRunningFalseAfterProcessExits(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 0"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}
	<-p.Done()

	if p.IsRunning() {
		t.Error("IsRunning() = true, want false for exited process")
	}
}

func TestStopGracefullyTerminatesRunningProcess(t *testing.T) {
	p := NewProcess("/bin/sleep")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"60"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	err = p.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() = %v, want nil", err)
	}

	select {
	case <-p.Done():
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("Done() was not closed after Stop()")
	}
}

func TestConcurrentDoneAndExitErrNoRace(t *testing.T) {
	p := NewProcess("/bin/sh")
	p.SetLogWriter(&bytes.Buffer{})

	err := p.Start(context.Background(), []string{"-c", "exit 0"})
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			p.Done()
		}()
		go func() {
			defer wg.Done()
			p.ExitErr()
		}()
		go func() {
			defer wg.Done()
			p.IsRunning()
		}()
	}
	wg.Wait()
}
