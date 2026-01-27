package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/d2verb/alpaca/internal/client"
	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/daemon"
	"github.com/d2verb/alpaca/internal/preset"
)

// Exit codes
const (
	exitSuccess         = 0
	exitError           = 1
	exitDaemonNotRuning = 2
	exitPresetNotFound  = 3
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

func isDaemonRunning(socketPath string) bool {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

type CLI struct {
	Start  StartCmd  `cmd:"" help:"Start the daemon"`
	Stop   StopCmd   `cmd:"" help:"Stop the daemon"`
	Status StatusCmd `cmd:"" help:"Show current status"`
	Run    RunCmd    `cmd:"" help:"Load a model with the specified preset"`
	Kill   KillCmd   `cmd:"" help:"Stop the currently running model"`
	Preset PresetCmd `cmd:"" help:"Manage presets"`
	Pull   PullCmd   `cmd:"" help:"Download model from HuggingFace"`

	Version VersionCmd `cmd:"" help:"Show version"`
}

type StartCmd struct{}

func (c *StartCmd) Run() error {
	paths := getPaths()

	// Check if already running
	if isDaemonRunning(paths.Socket) {
		fmt.Println("Daemon is already running.")
		return nil
	}

	// Create directories if needed
	if err := paths.EnsureDirectories(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	// Start daemon in foreground (MVP simplification)
	presetLoader := preset.NewLoader(paths.Presets)
	d := daemon.New(&daemon.Config{
		LlamaServerPath: "llama-server",
		SocketPath:      paths.Socket,
	}, presetLoader)

	server := daemon.NewServer(d, paths.Socket)

	fmt.Printf("Daemon listening on %s\n", paths.Socket)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	<-ctx.Done()
	if err := server.Stop(); err != nil {
		return fmt.Errorf("stop server: %w", err)
	}

	fmt.Println("\nDaemon stopped.")
	return nil
}

type StopCmd struct{}

func (c *StopCmd) Run() error {
	// For MVP (foreground daemon), user uses Ctrl+C
	fmt.Println("Use Ctrl+C to stop the daemon running in foreground.")
	return nil
}

type StatusCmd struct{}

func (c *StatusCmd) Run() error {
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

	return nil
}

type RunCmd struct {
	Preset string `arg:"" help:"Preset name to load"`
}

func (c *RunCmd) Run() error {
	cl := newClient()

	fmt.Printf("Loading %s...\n", c.Preset)
	resp, err := cl.Run(c.Preset)
	if err != nil {
		if strings.Contains(err.Error(), "connect") {
			fmt.Println("Daemon is not running. Run: alpaca start")
			os.Exit(exitDaemonNotRuning)
		}
		return fmt.Errorf("load preset: %w", err)
	}

	if resp.Status == "error" {
		if strings.Contains(resp.Error, "not found") {
			fmt.Printf("Preset '%s' not found.\n", c.Preset)
			os.Exit(exitPresetNotFound)
		}
		return fmt.Errorf("%s", resp.Error)
	}

	endpoint, _ := resp.Data["endpoint"].(string)
	fmt.Printf("Model ready at %s\n", endpoint)
	return nil
}

type KillCmd struct{}

func (c *KillCmd) Run() error {
	cl := newClient()
	resp, err := cl.Kill()
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
	List PresetListCmd `cmd:"" help:"List available presets"`
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

type PullCmd struct {
	Model string `arg:"" help:"Model to download (format: repo:quant)"`
}

func (c *PullCmd) Run() error {
	fmt.Printf("Pulling model: %s\n", c.Model)
	// TODO: Implement model download
	return nil
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
	)
	err := ctx.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitError)
	}
}
