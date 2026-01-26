package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

var version = "dev"

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
	fmt.Println("Starting daemon...")
	// TODO: Implement daemon start
	return nil
}

type StopCmd struct{}

func (c *StopCmd) Run() error {
	fmt.Println("Stopping daemon...")
	// TODO: Implement daemon stop
	return nil
}

type StatusCmd struct{}

func (c *StatusCmd) Run() error {
	fmt.Println("Checking status...")
	// TODO: Implement status check
	return nil
}

type RunCmd struct {
	Preset string `arg:"" help:"Preset name to load"`
}

func (c *RunCmd) Run() error {
	fmt.Printf("Loading preset: %s\n", c.Preset)
	// TODO: Implement model loading
	return nil
}

type KillCmd struct{}

func (c *KillCmd) Run() error {
	fmt.Println("Stopping model...")
	// TODO: Implement model stop
	return nil
}

type PresetCmd struct {
	List PresetListCmd `cmd:"" help:"List available presets"`
}

type PresetListCmd struct{}

func (c *PresetListCmd) Run() error {
	fmt.Println("Available presets:")
	// TODO: Implement preset listing
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
		os.Exit(1)
	}
}
