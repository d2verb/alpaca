// Package llama handles llama-server process management.
package llama

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const (
	// GracefulShutdownTimeout is the time to wait for graceful shutdown.
	GracefulShutdownTimeout = 10 * time.Second
)

// Process represents a llama-server process.
type Process struct {
	mu        sync.RWMutex
	path      string
	cmd       *exec.Cmd
	logWriter io.Writer
	done      chan struct{} // closed when process exits
	exitErr   error         // set before done is closed
}

// NewProcess creates a new process manager.
func NewProcess(path string) *Process {
	return &Process{path: path}
}

// SetLogWriter sets the log writer for llama-server output.
// If not set, stdout/stderr are used.
func (p *Process) SetLogWriter(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.logWriter = w
}

// Start starts the llama-server process with the given arguments.
// This is a non-blocking operation that forks the process and returns immediately.
// Use Stop() to manage the process lifecycle.
func (p *Process) Start(args []string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != nil && p.cmd.Process != nil {
		return fmt.Errorf("process already running")
	}

	p.cmd = exec.Command(p.path, args...)

	if p.logWriter != nil {
		p.cmd.Stdout = p.logWriter
		p.cmd.Stderr = p.logWriter
	} else {
		p.cmd.Stdout = os.Stdout
		p.cmd.Stderr = os.Stderr
	}

	if err := p.cmd.Start(); err != nil {
		return &ProcessError{Op: ProcessOpStart, Err: err}
	}

	p.done = make(chan struct{})
	go func() {
		err := p.cmd.Wait()
		p.mu.Lock()
		p.exitErr = err
		p.mu.Unlock()
		close(p.done)
	}()

	return nil
}

// Stop stops the llama-server process gracefully.
func (p *Process) Stop(ctx context.Context) error {
	p.mu.Lock()
	cmd := p.cmd
	done := p.done
	p.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	select {
	case <-done:
		return nil // already exited
	default:
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		select {
		case <-done:
			return nil
		default:
			return fmt.Errorf("send SIGTERM: %w", err)
		}
	}

	select {
	case <-done:
		return nil
	case <-time.After(GracefulShutdownTimeout):
		cmd.Process.Kill() // ignore error: process may have exited between timeout and kill
		<-done
		return nil
	case <-ctx.Done():
		cmd.Process.Kill() // ignore error: best-effort cleanup
		<-done
		return ctx.Err()
	}
}

// Done returns a channel that is closed when the process exits.
// Returns nil if the process has not been started.
func (p *Process) Done() <-chan struct{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.done
}

// ExitErr returns the error from the process exit.
// Only valid after Done() is closed.
func (p *Process) ExitErr() error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.exitErr
}

// IsRunning returns true if the process is running.
func (p *Process) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.done == nil {
		return false
	}
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}
