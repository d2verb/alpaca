package main

import (
	"errors"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/willabides/kongplete"

	"github.com/d2verb/alpaca/internal/ui"
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
	Open    OpenCmd    `cmd:"" help:"Open llama-server in browser"`
	Upgrade UpgradeCmd `cmd:"" help:"Upgrade alpaca to the latest version"`
	Version VersionCmd `cmd:"" help:"Show version"`

	// Completion commands
	CompletionScript kongplete.InstallCompletions `cmd:"" name:"completion-script" help:"Output shell completion script"`
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
				printExitError(exitErr)
			}
			os.Exit(exitErr.Code)
		}
		ui.PrintError(err.Error())
		os.Exit(exitError)
	}
}

func printExitError(e *ExitError) {
	lines := strings.Split(e.Message, "\n")
	if len(lines) == 0 {
		return
	}

	// Print first line with icon
	if e.Kind == ExitKindInfo {
		ui.PrintInfo(lines[0])
	} else {
		ui.PrintError(lines[0])
	}

	// Print remaining lines with indent
	for _, line := range lines[1:] {
		ui.PrintInfo(line)
	}
}
