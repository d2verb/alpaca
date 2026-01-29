package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/daemon"
	"github.com/d2verb/alpaca/internal/logging"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/ui"
)

type StartCmd struct {
	Foreground bool `short:"f" help:"Run in foreground (don't daemonize)"`
}

func (c *StartCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	// Check if already running
	status, err := daemon.GetDaemonStatus(paths.PID, paths.Socket)
	if err != nil && !errors.Is(err, daemon.ErrPIDFileNotFound) {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if status.Running {
		ui.PrintInfo(fmt.Sprintf("Daemon is already running (PID: %d)", status.PID))
		return nil
	}

	// Clean up stale files if any
	if status.SocketExists && !status.Running {
		ui.PrintWarning("Cleaning up stale socket...")
		os.Remove(paths.Socket)
	}
	if status.PID > 0 && !status.Running {
		daemon.RemovePIDFile(paths.PID)
	}

	// Create directories if needed
	if err := paths.EnsureDirectories(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	// If not foreground mode, spawn background process
	if !c.Foreground {
		return c.startBackground(paths)
	}

	// Foreground mode: run the actual daemon
	return c.runDaemon(paths)
}

func (c *StartCmd) startBackground(paths *config.Paths) error {
	// Re-exec ourselves with --foreground flag
	cmd := exec.Command(os.Args[0], "start", "--foreground")
	cmd.Env = os.Environ()

	// Detach from controlling terminal (Unix/macOS only)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session and detach from terminal
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	// Wait for daemon to become ready (max 5 seconds)
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if daemon.IsSocketAvailable(paths.Socket) {
			// User-facing output (not logged)
			ui.PrintSuccess(fmt.Sprintf("Daemon started (PID: %d)", cmd.Process.Pid))
			ui.PrintInfo(fmt.Sprintf("Logs: %s", paths.DaemonLog))
			return nil
		}
	}

	return fmt.Errorf("daemon did not start within 5 seconds, check logs: %s", paths.DaemonLog)
}

func (c *StartCmd) runDaemon(paths *config.Paths) error {
	// Set up log writers
	daemonLogWriter := logging.NewRotatingWriter(logging.DefaultConfig(paths.DaemonLog))
	defer daemonLogWriter.Close()

	llamaLogWriter := logging.NewRotatingWriter(logging.DefaultConfig(paths.LlamaLog))
	defer llamaLogWriter.Close()

	// Create logger for daemon
	logger := logging.NewLogger(daemonLogWriter)
	logger.Info("daemon starting")

	// Write PID file
	if err := daemon.WritePIDFile(paths.PID); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	defer daemon.RemovePIDFile(paths.PID)

	// Load user config
	userConfig, err := config.LoadConfig(paths.Config)
	if err != nil {
		userConfig = config.DefaultConfig()
	}

	// Start daemon
	presetLoader := preset.NewLoader(paths.Presets)
	modelManager := model.NewManager(paths.Models)
	d := daemon.New(&daemon.Config{
		LlamaServerPath: userConfig.LlamaServerPath,
		SocketPath:      paths.Socket,
		LlamaLogWriter:  llamaLogWriter,
	}, presetLoader, modelManager, userConfig)

	server := daemon.NewServer(d, paths.Socket)

	logger.Info("daemon listening", "socket", paths.Socket)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		logger.Error("start server failed", "error", err)
		return fmt.Errorf("start server: %w", err)
	}

	<-ctx.Done()
	logger.Info("daemon stopping")

	if err := server.Stop(); err != nil {
		logger.Error("stop server failed", "error", err)
		return fmt.Errorf("stop server: %w", err)
	}

	logger.Info("daemon stopped")
	return nil
}
