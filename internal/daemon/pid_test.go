package daemon

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWritePIDFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "test.pid")

	// Act
	err := WritePIDFile(pidPath)

	// Assert
	if err != nil {
		t.Fatalf("WritePIDFile failed: %v", err)
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	if len(data) == 0 {
		t.Error("PID file is empty")
	}
}

func TestReadPIDFile(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantPID   int
		wantError error
	}{
		{
			name:      "valid PID",
			content:   "12345",
			wantPID:   12345,
			wantError: nil,
		},
		{
			name:      "valid PID with whitespace",
			content:   "  12345  \n",
			wantPID:   12345,
			wantError: nil,
		},
		{
			name:      "invalid PID - not a number",
			content:   "not-a-number",
			wantPID:   0,
			wantError: ErrInvalidPIDFile,
		},
		{
			name:      "invalid PID - zero",
			content:   "0",
			wantPID:   0,
			wantError: ErrInvalidPIDFile,
		},
		{
			name:      "invalid PID - negative",
			content:   "-123",
			wantPID:   0,
			wantError: ErrInvalidPIDFile,
		},
		{
			name:      "empty file",
			content:   "",
			wantPID:   0,
			wantError: ErrInvalidPIDFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			tmpDir := t.TempDir()
			pidPath := filepath.Join(tmpDir, "test.pid")
			if err := os.WriteFile(pidPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Act
			pid, err := ReadPIDFile(pidPath)

			// Assert
			if tt.wantError != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantError)
				} else if !errors.Is(err, tt.wantError) {
					t.Errorf("Expected error %v, got %v", tt.wantError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if pid != tt.wantPID {
				t.Errorf("Expected PID %d, got %d", tt.wantPID, pid)
			}
		})
	}
}

func TestReadPIDFileNotFound(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "nonexistent.pid")

	// Act
	pid, err := ReadPIDFile(pidPath)

	// Assert
	if !errors.Is(err, ErrPIDFileNotFound) {
		t.Errorf("Expected ErrPIDFileNotFound, got %v", err)
	}
	if pid != 0 {
		t.Errorf("Expected PID 0, got %d", pid)
	}
}

func TestIsProcessRunning(t *testing.T) {
	tests := []struct {
		name        string
		pid         int
		wantRunning bool
		wantError   bool
	}{
		{
			name:        "current process",
			pid:         os.Getpid(),
			wantRunning: true,
			wantError:   false,
		},
		{
			name:        "init process (PID 1)",
			pid:         1,
			wantRunning: true,
			wantError:   false,
		},
		{
			name:        "invalid PID - zero",
			pid:         0,
			wantRunning: false,
			wantError:   true,
		},
		{
			name:        "invalid PID - negative",
			pid:         -1,
			wantRunning: false,
			wantError:   true,
		},
		{
			name:        "non-existent process",
			pid:         99999999,
			wantRunning: false,
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			running, err := IsProcessRunning(tt.pid)

			// Assert
			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if running != tt.wantRunning {
				t.Errorf("Expected running=%v, got %v", tt.wantRunning, running)
			}
		})
	}
}

func TestIsSocketAvailable(t *testing.T) {
	t.Run("available socket", func(t *testing.T) {
		// Arrange
		// Use /tmp to avoid Unix socket path length limits (108 chars)
		socketPath := filepath.Join("/tmp", "alpaca-test.sock")
		defer os.Remove(socketPath)

		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to create socket: %v", err)
		}
		defer listener.Close()

		// Act
		available := IsSocketAvailable(socketPath)

		// Assert
		if !available {
			t.Error("Expected socket to be available")
		}
	})

	t.Run("non-existent socket", func(t *testing.T) {
		// Arrange
		socketPath := filepath.Join("/tmp", "alpaca-nonexistent.sock")

		// Act
		available := IsSocketAvailable(socketPath)

		// Assert
		if available {
			t.Error("Expected socket to be unavailable")
		}
	})
}

func TestGetDaemonStatus(t *testing.T) {
	t.Run("daemon running", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		pidPath := filepath.Join(tmpDir, "test.pid")
		socketPath := filepath.Join("/tmp", "alpaca-test-status.sock")
		defer os.Remove(socketPath)

		// Create PID file with current process
		if err := WritePIDFile(pidPath); err != nil {
			t.Fatalf("Failed to write PID file: %v", err)
		}

		// Create socket
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to create socket: %v", err)
		}
		defer listener.Close()

		// Act
		status, err := GetDaemonStatus(pidPath, socketPath)

		// Assert
		if err != nil {
			t.Fatalf("GetDaemonStatus failed: %v", err)
		}

		if !status.Running {
			t.Error("Expected daemon to be running")
		}

		if !status.SocketExists {
			t.Error("Expected socket to exist")
		}

		if status.PID != os.Getpid() {
			t.Errorf("Expected PID %d, got %d", os.Getpid(), status.PID)
		}
	})

	t.Run("daemon not running - no PID file", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		pidPath := filepath.Join(tmpDir, "test.pid")
		socketPath := filepath.Join(tmpDir, "test.sock")

		// Act
		status, err := GetDaemonStatus(pidPath, socketPath)

		// Assert
		if err != nil {
			t.Fatalf("GetDaemonStatus failed: %v", err)
		}

		if status.Running {
			t.Error("Expected daemon to not be running")
		}

		if status.SocketExists {
			t.Error("Expected socket to not exist")
		}

		if status.PID != 0 {
			t.Errorf("Expected PID 0, got %d", status.PID)
		}
	})

	t.Run("stale socket - process not running", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		pidPath := filepath.Join(tmpDir, "test.pid")
		socketPath := filepath.Join("/tmp", "alpaca-test-stale.sock")
		defer os.Remove(socketPath)

		// Create PID file with non-existent process
		if err := os.WriteFile(pidPath, []byte("99999999"), 0644); err != nil {
			t.Fatalf("Failed to write PID file: %v", err)
		}

		// Create socket
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatalf("Failed to create socket: %v", err)
		}
		defer listener.Close()

		// Act
		status, err := GetDaemonStatus(pidPath, socketPath)

		// Assert
		if err == nil {
			t.Error("Expected error for stale socket")
		}

		if status.Running {
			t.Error("Expected daemon to not be running")
		}

		if !status.SocketExists {
			t.Error("Expected socket to exist")
		}
	})

	t.Run("invalid PID file", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		pidPath := filepath.Join(tmpDir, "test.pid")
		socketPath := filepath.Join(tmpDir, "test.sock")

		// Create invalid PID file
		if err := os.WriteFile(pidPath, []byte("invalid"), 0644); err != nil {
			t.Fatalf("Failed to write PID file: %v", err)
		}

		// Act
		status, err := GetDaemonStatus(pidPath, socketPath)

		// Assert
		if err == nil {
			t.Error("Expected error for invalid PID file")
		}

		if !errors.Is(err, ErrInvalidPIDFile) {
			t.Errorf("Expected ErrInvalidPIDFile, got %v", err)
		}

		if status.Running {
			t.Error("Expected daemon to not be running")
		}
	})
}

func TestRemovePIDFile(t *testing.T) {
	t.Run("remove existing file", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		pidPath := filepath.Join(tmpDir, "test.pid")
		if err := WritePIDFile(pidPath); err != nil {
			t.Fatalf("Failed to write PID file: %v", err)
		}

		// Act
		err := RemovePIDFile(pidPath)

		// Assert
		if err != nil {
			t.Errorf("RemovePIDFile failed: %v", err)
		}

		if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
			t.Error("Expected PID file to be removed")
		}
	})

	t.Run("remove non-existent file", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		pidPath := filepath.Join(tmpDir, "nonexistent.pid")

		// Act
		err := RemovePIDFile(pidPath)

		// Assert
		if err != nil {
			t.Errorf("RemovePIDFile should not fail for non-existent file: %v", err)
		}
	})
}

// TestProcessSignalZero verifies that signal 0 works as expected on this platform.
func TestProcessSignalZero(t *testing.T) {
	// Arrange
	currentPID := os.Getpid()
	process, err := os.FindProcess(currentPID)
	if err != nil {
		t.Fatalf("FindProcess failed: %v", err)
	}

	// Act
	err = process.Signal(syscall.Signal(0))

	// Assert
	if err != nil {
		t.Errorf("Signal 0 to current process should succeed, got: %v", err)
	}

	// Test with non-existent process
	nonExistentPID := 99999999
	process, err = os.FindProcess(nonExistentPID)
	if err != nil {
		t.Fatalf("FindProcess failed: %v", err)
	}

	err = process.Signal(syscall.Signal(0))
	if err == nil {
		t.Error("Signal 0 to non-existent process should fail")
	}

	if !errors.Is(err, syscall.ESRCH) {
		t.Logf("Note: Expected ESRCH, got %v (may vary by platform)", err)
	}
}
