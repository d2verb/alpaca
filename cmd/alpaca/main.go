package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/willabides/kongplete"
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
	List    ListCmd    `cmd:"" name:"ls" help:"List presets and models"`
	Show    ShowCmd    `cmd:"" help:"Show details of a preset or model"`
	Remove  RemoveCmd  `cmd:"" name:"rm" help:"Remove a preset or model"`
	Pull    PullCmd    `cmd:"" help:"Download a model"`
	New     NewCmd     `cmd:"" help:"Create a new preset interactively"`
	Version VersionCmd `cmd:"" help:"Show version"`

	// Completion commands
	InstallCompletions kongplete.InstallCompletions `cmd:"" help:"Install shell completions"`
}

func main() {
	cli := CLI{}
	parser, err := kong.New(&cli,
		kong.Name("alpaca"),
		kong.Description("Lightweight llama-server wrapper"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	if err != nil {
		panic(err)
	}

	// Add completion support with different predictors per command
	kongplete.Complete(parser,
		kongplete.WithPredictor("show-identifier", newShowIdentifierPredictor()),
		kongplete.WithPredictor("rm-identifier", newRmIdentifierPredictor()),
		kongplete.WithPredictor("load-identifier", newLoadIdentifierPredictor()),
	)

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		parser.FatalIfErrorf(err)
	}

	err = ctx.Run()
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
