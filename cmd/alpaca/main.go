package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

var (
	version = "dev"
	commit  = "unknown"
)

type CLI struct {
	Start   StartCmd   `cmd:"" help:"Start the daemon"`
	Stop    StopCmd    `cmd:"" help:"Stop the daemon"`
	Status  StatusCmd  `cmd:"" help:"Show current status"`
	Load    LoadCmd    `cmd:"" help:"Load a preset, model, or file"`
	Unload  UnloadCmd  `cmd:"" help:"Stop the currently running model"`
	Logs    LogsCmd    `cmd:"" help:"Show logs (daemon or server)"`
	Preset  PresetCmd  `cmd:"" help:"Manage presets"`
	Model   ModelCmd   `cmd:"" help:"Manage models"`
	Version VersionCmd `cmd:"" help:"Show version"`
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
		var exitErr *ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Message != "" {
				fmt.Fprintln(os.Stderr, exitErr.Message)
			}
			os.Exit(exitErr.Code)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(exitError)
	}
}
