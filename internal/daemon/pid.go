// Package daemon provides daemon process management utilities.
package daemon

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

var (
	// ErrPIDFileNotFound is returned when the PID file does not exist.
	ErrPIDFileNotFound = errors.New("PID file not found")
	// ErrInvalidPIDFile is returned when the PID file contains invalid data.
	ErrInvalidPIDFile = errors.New("invalid PID file")
	// ErrProcessNotFound is returned when the process does not exist.
	ErrProcessNotFound = errors.New("process not found")
)

// DaemonStatus represents the current status of the daemon.
type DaemonStatus struct {
	Running      bool
	PID          int
	SocketExists bool
}

// WritePIDFile writes the current process ID to the specified file.
func WritePIDFile(path string) error {
	pid := os.Getpid()
	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	return nil
}

// ReadPIDFile reads the process ID from the specified file.
// Returns ErrPIDFileNotFound if the file doesn't exist.
// Returns ErrInvalidPIDFile if the file contains invalid data.
func ReadPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrPIDFileNotFound
		}
		return 0, fmt.Errorf("read PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidPIDFile, err)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("%w: invalid PID %d", ErrInvalidPIDFile, pid)
	}

	return pid, nil
}

// IsProcessRunning checks if a process with the given PID is running.
// On Unix/macOS, it uses signal 0 to check process existence without killing it.
func IsProcessRunning(pid int) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("invalid PID: %d", pid)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("find process: %w", err)
	}

	// Signal 0 checks if we can send a signal without actually sending it
	// This is a Unix/macOS-specific way to check if a process exists
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true, nil
	}

	// ESRCH means the process doesn't exist
	if errors.Is(err, syscall.ESRCH) {
		return false, nil
	}

	// EPERM means the process exists but we don't have permission
	if errors.Is(err, syscall.EPERM) {
		return true, nil
	}

	// "process already finished" is returned on macOS for non-existent processes
	if err.Error() == "os: process already finished" {
		return false, nil
	}

	return false, fmt.Errorf("check process: %w", err)
}

// IsSocketAvailable checks if the daemon socket is accessible.
func IsSocketAvailable(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetDaemonStatus checks the daemon status by examining both the PID file
// and socket availability. This provides a unified view of daemon state.
func GetDaemonStatus(pidPath, socketPath string) (*DaemonStatus, error) {
	status := &DaemonStatus{}

	// Check socket first (quick check)
	status.SocketExists = IsSocketAvailable(socketPath)

	// Read PID file
	pid, err := ReadPIDFile(pidPath)
	if err != nil {
		if errors.Is(err, ErrPIDFileNotFound) {
			// No PID file, daemon not running
			return status, nil
		}
		// Invalid PID file, might be corrupted
		return status, fmt.Errorf("read PID: %w", err)
	}

	status.PID = pid

	// Check if process is running
	running, err := IsProcessRunning(pid)
	if err != nil {
		return status, fmt.Errorf("check process %d: %w", pid, err)
	}

	status.Running = running

	// Consistency check: if socket exists but process not running,
	// or if process running but socket doesn't exist, something is wrong
	if status.SocketExists && !status.Running {
		return status, fmt.Errorf("socket exists but process %d not running (stale socket?)", pid)
	}

	return status, nil
}

// RemovePIDFile removes the PID file. It's safe to call even if the file doesn't exist.
func RemovePIDFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove PID file: %w", err)
	}
	return nil
}
