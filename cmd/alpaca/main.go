package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/d2verb/alpaca/internal/client"
	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/daemon"
	"github.com/d2verb/alpaca/internal/logging"
	"github.com/d2verb/alpaca/internal/model"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/pull"
)

// Exit codes
const (
	exitSuccess         = 0
	exitError           = 1
	exitDaemonNotRuning = 2
	exitPresetNotFound  = 3
	exitModelNotFound   = 4
	exitDownloadFailed  = 5
)

var version = "dev"

// Helper functions

func getPaths() *config.Paths {
	paths, err := config.GetPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get paths: %v\n", err)
		os.Exit(exitError)
	}
	return paths
}

func newClient() *client.Client {
	paths := getPaths()
	return client.New(paths.Socket)
}

type CLI struct {
	Start   StartCmd   `cmd:"" help:"Start the daemon"`
	Stop    StopCmd    `cmd:"" help:"Stop the daemon"`
	Status  StatusCmd  `cmd:"" help:"Show current status"`
	Load    LoadCmd    `cmd:"" help:"Load a model (preset or HuggingFace format)"`
	Unload  UnloadCmd  `cmd:"" help:"Stop the currently running model"`
	Preset  PresetCmd  `cmd:"" help:"Manage presets"`
	Model   ModelCmd   `cmd:"" help:"Manage models"`
	Version VersionCmd `cmd:"" help:"Show version"`
}

type StartCmd struct {
	Foreground bool `short:"f" help:"Run in foreground (don't daemonize)"`
}

func (c *StartCmd) Run() error {
	paths := getPaths()

	// Check if already running
	status, err := daemon.GetDaemonStatus(paths.PID, paths.Socket)
	if err != nil && !errors.Is(err, daemon.ErrPIDFileNotFound) {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if status.Running {
		fmt.Printf("Daemon is already running (PID: %d).\n", status.PID)
		return nil
	}

	// Clean up stale files if any
	if status.SocketExists && !status.Running {
		fmt.Println("Cleaning up stale socket...")
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
			fmt.Printf("Daemon started (PID: %d)\n", cmd.Process.Pid)
			fmt.Printf("Logs: %s\n", paths.DaemonLog)
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

type StopCmd struct{}

func (c *StopCmd) Run() error {
	paths := getPaths()

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
		fmt.Println("Daemon is not running.")
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

	fmt.Println("Stopping daemon...")
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
			fmt.Println("Daemon stopped.")
			daemon.RemovePIDFile(paths.PID)
			return nil
		}
	}

	// Force kill if still running
	fmt.Println("Daemon did not stop gracefully, forcing...")
	if err := process.Kill(); err != nil {
		return fmt.Errorf("kill daemon: %w", err)
	}

	daemon.RemovePIDFile(paths.PID)
	fmt.Println("Daemon stopped.")
	return nil
}

type StatusCmd struct{}

func (c *StatusCmd) Run() error {
	paths := getPaths()
	cl := newClient()
	resp, err := cl.Status()
	if err != nil {
		fmt.Println("Daemon is not running.")
		fmt.Println("Run: alpaca start")
		os.Exit(exitDaemonNotRuning)
	}

	state, _ := resp.Data["state"].(string)
	fmt.Printf("Status: %s\n", state)

	if presetName, ok := resp.Data["preset"].(string); ok {
		fmt.Printf("Preset: %s\n", presetName)
	}
	if endpoint, ok := resp.Data["endpoint"].(string); ok {
		fmt.Printf("Endpoint: %s\n", endpoint)
	}
	fmt.Printf("Logs: %s\n", paths.DaemonLog)

	return nil
}

type LoadCmd struct {
	Identifier string `arg:"" help:"Preset name or HuggingFace format (repo:quant)"`
}

func (c *LoadCmd) Run() error {
	paths := getPaths()
	cl := newClient()
	ctx := context.Background()

	// If HF format, check if downloaded
	if strings.Contains(c.Identifier, ":") {
		repo, quant, err := pull.ParseModelSpec(c.Identifier)
		if err != nil {
			return fmt.Errorf("invalid model spec: %w", err)
		}

		// Check if model exists
		modelMgr := model.NewManager(paths.Models)
		exists, err := modelMgr.Exists(ctx, repo, quant)
		if err != nil {
			return fmt.Errorf("check model: %w", err)
		}

		// Auto-pull if not downloaded
		if !exists {
			fmt.Println("Model not found. Downloading...")
			if err := pullModel(repo, quant, paths.Models); err != nil {
				os.Exit(exitDownloadFailed)
			}
		}
	}

	// Load model
	fmt.Printf("Loading %s...\n", c.Identifier)
	resp, err := cl.Load(c.Identifier)
	if err != nil {
		if strings.Contains(err.Error(), "connect") {
			fmt.Println("Daemon is not running. Run: alpaca start")
			os.Exit(exitDaemonNotRuning)
		}
		return fmt.Errorf("load model: %w", err)
	}

	if resp.Status == "error" {
		if strings.Contains(resp.Error, "not found") {
			fmt.Printf("Model '%s' not found.\n", c.Identifier)
			os.Exit(exitModelNotFound)
		}
		return fmt.Errorf("%s", resp.Error)
	}

	endpoint, _ := resp.Data["endpoint"].(string)
	fmt.Printf("Model ready at %s\n", endpoint)
	return nil
}

type UnloadCmd struct{}

func (c *UnloadCmd) Run() error {
	cl := newClient()
	resp, err := cl.Unload()
	if err != nil {
		fmt.Println("Daemon is not running.")
		os.Exit(exitDaemonNotRuning)
	}

	if resp.Status == "error" {
		return fmt.Errorf("%s", resp.Error)
	}

	fmt.Println("Model stopped.")
	return nil
}

type PresetCmd struct {
	List   PresetListCmd `cmd:"" name:"ls" help:"List available presets"`
	Remove PresetRmCmd   `cmd:"" name:"rm" help:"Remove a preset"`
}

type PresetListCmd struct{}

func (c *PresetListCmd) Run() error {
	cl := newClient()
	resp, err := cl.ListPresets()
	if err != nil {
		fmt.Println("Daemon is not running.")
		os.Exit(exitDaemonNotRuning)
	}

	if resp.Status == "error" {
		return fmt.Errorf("%s", resp.Error)
	}

	presets, _ := resp.Data["presets"].([]any)
	if len(presets) == 0 {
		fmt.Println("No presets available.")
		fmt.Printf("Add presets to: %s\n", getPaths().Presets)
		return nil
	}

	fmt.Println("Available presets:")
	for _, p := range presets {
		fmt.Printf("  - %s\n", p.(string))
	}
	return nil
}

type PresetRmCmd struct {
	Name string `arg:"" help:"Preset name to remove"`
}

func (c *PresetRmCmd) Run() error {
	paths := getPaths()
	presetPath := fmt.Sprintf("%s/%s.yaml", paths.Presets, c.Name)

	// Check if preset exists
	if _, err := os.Stat(presetPath); os.IsNotExist(err) {
		fmt.Printf("Preset '%s' not found.\n", c.Name)
		os.Exit(exitPresetNotFound)
	}

	// Confirmation prompt
	fmt.Printf("Delete preset '%s'? (y/N): ", c.Name)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete file
	if err := os.Remove(presetPath); err != nil {
		return fmt.Errorf("remove preset: %w", err)
	}

	fmt.Printf("Preset '%s' removed.\n", c.Name)
	return nil
}

type ModelCmd struct {
	List   ModelListCmd `cmd:"" name:"ls" help:"List downloaded models"`
	Pull   ModelPullCmd `cmd:"" help:"Download a model"`
	Remove ModelRmCmd   `cmd:"" name:"rm" help:"Remove a model"`
}

type ModelListCmd struct{}

func (c *ModelListCmd) Run() error {
	paths := getPaths()
	modelMgr := model.NewManager(paths.Models)
	ctx := context.Background()

	entries, err := modelMgr.List(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No models downloaded.")
		fmt.Println("Run: alpaca model pull <repo>:<quant>")
		return nil
	}

	fmt.Println("Downloaded models:")
	for _, entry := range entries {
		fmt.Printf("  - %s:%s (%s)\n", entry.Repo, entry.Quant, formatSize(entry.Size))
	}
	return nil
}

type ModelPullCmd struct {
	Model string `arg:"" help:"Model to download (format: repo:quant)"`
}

func (c *ModelPullCmd) Run() error {
	repo, quant, err := pull.ParseModelSpec(c.Model)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Println("Format: alpaca model pull <org>/<repo>:<quant>")
		fmt.Println("Example: alpaca model pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
		os.Exit(exitError)
	}

	paths := getPaths()
	if err := pullModel(repo, quant, paths.Models); err != nil {
		os.Exit(exitDownloadFailed)
	}
	return nil
}

type ModelRmCmd struct {
	Model string `arg:"" help:"Model to remove (format: repo:quant)"`
}

func (c *ModelRmCmd) Run() error {
	repo, quant, err := pull.ParseModelSpec(c.Model)
	if err != nil {
		return fmt.Errorf("invalid model spec: %w", err)
	}

	paths := getPaths()
	modelMgr := model.NewManager(paths.Models)
	ctx := context.Background()

	// Check if model exists
	exists, err := modelMgr.Exists(ctx, repo, quant)
	if err != nil {
		return fmt.Errorf("check model: %w", err)
	}
	if !exists {
		fmt.Printf("Model '%s:%s' not found.\n", repo, quant)
		os.Exit(exitModelNotFound)
	}

	// Confirmation prompt
	fmt.Printf("Delete model '%s:%s'? (y/N): ", repo, quant)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Remove model
	if err := modelMgr.Remove(ctx, repo, quant); err != nil {
		return fmt.Errorf("remove model: %w", err)
	}

	fmt.Printf("Model '%s:%s' removed.\n", repo, quant)
	return nil
}

// pullModel downloads a model from HuggingFace.
func pullModel(repo, quant, modelsDir string) error {
	paths := getPaths()
	if err := paths.EnsureDirectories(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	puller := pull.NewPuller(modelsDir)

	// Get file info first
	fmt.Println("Fetching file list...")
	filename, size, err := puller.GetFileInfo(context.Background(), repo, quant)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return err
	}

	fmt.Printf("Downloading %s (%s)...\n", filename, formatSize(size))

	// Set up progress reporting
	puller.SetProgressFunc(func(downloaded, total int64) {
		printProgress(downloaded, total)
	})

	// Download
	result, err := puller.Pull(context.Background(), repo, quant)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		return err
	}

	// Ensure progress bar shows 100% completion
	if result.Size > 0 {
		printProgress(result.Size, result.Size)
	}
	fmt.Printf("\nSaved to: %s\n", result.Path)
	return nil
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func printProgress(downloaded, total int64) {
	if total <= 0 {
		fmt.Printf("\r%s downloaded", formatSize(downloaded))
		return
	}

	percent := float64(downloaded) / float64(total) * 100
	barWidth := 40
	filled := int(percent / 100 * float64(barWidth))

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("\r[%s] %.1f%% (%s / %s)", bar, percent, formatSize(downloaded), formatSize(total))
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Printf("alpaca version %s\n", version)
	return nil
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("alpaca"),
		kong.Description("Lightweight llama-server wrapper"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	err := ctx.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitError)
	}
}
