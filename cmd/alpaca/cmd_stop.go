package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/d2verb/alpaca/internal/daemon"
	"github.com/d2verb/alpaca/internal/ui"
)

type StopCmd struct{}

func (c *StopCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	// Get daemon status
	status, err := daemon.GetDaemonStatus(paths.PID, paths.Socket)
	if err != nil && !errors.Is(err, daemon.ErrPIDFileNotFound) {
		// If there's an error but socket exists, try to clean up
		if status.SocketExists {
			fmt.Println("Warning: stale daemon state detected")
			fmt.Printf("Manual cleanup may be needed: rm %s\n", paths.Socket)
		}
		return fmt.Errorf("check daemon status: %w", err)
	}

	if !status.Running {
		ui.PrintInfo("Daemon is not running")
		// Clean up stale files
		daemon.RemovePIDFile(paths.PID)
		if status.SocketExists {
			os.Remove(paths.Socket)
		}
		return nil
	}

	// Send SIGTERM
	process, err := os.FindProcess(status.PID)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	ui.PrintInfo("Stopping daemon...")
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	// Wait for process to exit (max 10 seconds)
	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		running, err := daemon.IsProcessRunning(status.PID)
		if err != nil {
			return fmt.Errorf("check process: %w", err)
		}
		if !running {
			ui.PrintSuccess("Daemon stopped")
			daemon.RemovePIDFile(paths.PID)
			return nil
		}
	}

	// Force kill if still running
	ui.PrintWarning("Daemon did not stop gracefully, forcing...")
	if err := process.Kill(); err != nil {
		return fmt.Errorf("kill daemon: %w", err)
	}

	daemon.RemovePIDFile(paths.PID)
	ui.PrintSuccess("Daemon stopped")
	return nil
}
