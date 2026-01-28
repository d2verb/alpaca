// Package llama handles llama-server process management.
package llama

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	// GracefulShutdownTimeout is the time to wait for graceful shutdown.
	GracefulShutdownTimeout = 10 * time.Second
)

// Process represents a llama-server process.
type Process struct {
	path      string
	cmd       *exec.Cmd
	logWriter io.Writer
}

// NewProcess creates a new process manager.
func NewProcess(path string) *Process {
	return &Process{path: path}
}

// SetLogWriter sets the log writer for llama-server output.
// If not set, stdout/stderr are used.
func (p *Process) SetLogWriter(w io.Writer) {
	p.logWriter = w
}

// Start starts the llama-server process with the given arguments.
func (p *Process) Start(ctx context.Context, args []string) error {
	p.cmd = exec.CommandContext(ctx, p.path, args...)

	if p.logWriter != nil {
		p.cmd.Stdout = p.logWriter
		p.cmd.Stderr = p.logWriter
	} else {
		p.cmd.Stdout = os.Stdout
		p.cmd.Stderr = os.Stderr
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("start llama-server: %w", err)
	}

	return nil
}

// Stop stops the llama-server process gracefully.
func (p *Process) Stop(ctx context.Context) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		_, err := p.cmd.Process.Wait()
		done <- err
	}()

	select {
	case <-done:
		// Process exited gracefully
		return nil
	case <-time.After(GracefulShutdownTimeout):
		// Timeout, force kill
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill llama-server: %w", err)
		}
		return nil
	case <-ctx.Done():
		// Context cancelled, force kill
		p.cmd.Process.Kill()
		return ctx.Err()
	}
}

// IsRunning returns true if the process is running.
func (p *Process) IsRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}
	// Check if process is still running
	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}
