package main

import (
	"fmt"
	"os"
	"syscall"
)

type LogsCmd struct {
	Follow bool `short:"f" help:"Follow log output in real-time (tail -f)"`
	Daemon bool `short:"d" help:"Show daemon logs (default)"`
	Server bool `short:"s" help:"Show llama-server logs"`
}

func (c *LogsCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	logPath := paths.DaemonLog
	if c.Server {
		logPath = paths.LlamaLog
	}

	// Check if log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s\nHint: Start the daemon first with 'alpaca start'", logPath)
	}

	// Build tail arguments
	args := []string{"tail"}
	if c.Follow {
		args = append(args, "-f")
	}
	args = append(args, logPath)

	// Find tail binary
	tailPath := "/usr/bin/tail"
	if _, err := os.Stat(tailPath); os.IsNotExist(err) {
		return fmt.Errorf("tail command not found at %s", tailPath)
	}

	// Replace current process with tail
	return syscall.Exec(tailPath, args, os.Environ())
}
