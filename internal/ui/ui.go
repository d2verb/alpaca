// Package ui provides formatted output utilities for the CLI.
package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// Color functions for consistent styling.
var (
	Green  = color.New(color.FgGreen).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
	Blue   = color.New(color.FgBlue).SprintFunc()
	Cyan   = color.New(color.FgCyan).SprintFunc()
	Dim    = color.New(color.Faint).SprintFunc() // Dimmed text (more readable than gray)
	Bold   = color.New(color.Bold).SprintFunc()
)

// Output is the destination for UI output.
// Defaults to os.Stdout but can be overridden for testing.
var Output io.Writer = os.Stdout

// FormatSize formats size string with dim color.
func FormatSize(size string) string {
	return Dim(size)
}

// FormatEndpoint formats endpoint with blue color.
func FormatEndpoint(endpoint string) string {
	return Blue(endpoint)
}

// StatusBadge returns a colored status indicator with label.
func StatusBadge(state string) string {
	switch state {
	case "running":
		return Green("● Running")
	case "loading":
		return Yellow("◐ Loading")
	case "idle":
		return Yellow("○ Idle")
	default:
		return Red("○ Not Running")
	}
}

// PrintStatus prints daemon status in a formatted style.
func PrintStatus(state, preset, endpoint, logPath string) {
	fmt.Fprintf(Output, "%s %s\n", Bold("Status:"), StatusBadge(state))

	if preset != "" {
		fmt.Fprintf(Output, "%s %s\n", Bold("Preset:"), preset)
	}

	if endpoint != "" {
		fmt.Fprintf(Output, "%s %s\n", Bold("Endpoint:"), Blue(endpoint))
	}

	fmt.Fprintf(Output, "%s %s\n", Bold("Logs:"), logPath)
}

// PrintModelList prints a list of downloaded models with formatting.
func PrintModelList(models []ModelInfo) {
	if len(models) == 0 {
		fmt.Fprintln(Output, "No models downloaded.")
		return
	}

	fmt.Fprintln(Output, Bold("Downloaded models:"))
	for _, m := range models {
		// Format: h:repo:quant (size)
		fmt.Fprintf(Output, "  %s:%s:%s %s\n",
			Cyan("h"),
			Cyan(m.Repo),
			Yellow(m.Quant),
			Dim(fmt.Sprintf("(%s)", m.SizeString)),
		)
	}
}

// ModelInfo represents a downloaded model for display.
type ModelInfo struct {
	Repo       string
	Quant      string
	SizeString string
}

// PrintPresetList prints a list of available presets with formatting.
func PrintPresetList(presets []string) {
	if len(presets) == 0 {
		fmt.Fprintln(Output, "No presets available.")
		return
	}

	fmt.Fprintln(Output, Bold("Available presets:"))
	for _, p := range presets {
		fmt.Fprintf(Output, "  %s:%s\n", Cyan("p"), Cyan(p))
	}
}

// PrintSuccess prints a success message with green checkmark.
func PrintSuccess(message string) {
	fmt.Fprintf(Output, "%s %s\n", Green("✓"), message)
}

// PrintError prints an error message with red X.
func PrintError(message string) {
	fmt.Fprintf(Output, "%s %s\n", Red("✗"), message)
}

// PrintWarning prints a warning message with yellow exclamation.
func PrintWarning(message string) {
	fmt.Fprintf(Output, "%s %s\n", Yellow("⚠"), message)
}

// PrintInfo prints an info message with blue dot.
func PrintInfo(message string) {
	fmt.Fprintf(Output, "%s %s\n", Blue("•"), message)
}

// PresetDetails contains preset information for display.
type PresetDetails struct {
	Name        string
	Model       string
	ContextSize int
	GPULayers   int
	Threads     int
	Host        string
	Port        int
	ExtraArgs   []string
}

// ModelDetails contains model metadata for display.
type ModelDetails struct {
	Repo         string
	Quant        string
	Filename     string
	Path         string
	Size         string
	DownloadedAt string
}

// PrintPresetDetails prints preset details in a formatted style.
func PrintPresetDetails(p PresetDetails) {
	fmt.Fprintf(Output, "%s %s\n", Bold("Name:"), Cyan(p.Name))
	fmt.Fprintf(Output, "%s %s\n", Bold("Model:"), p.Model)

	if p.ContextSize > 0 {
		fmt.Fprintf(Output, "%s %d\n", Bold("Context Size:"), p.ContextSize)
	}
	if p.GPULayers != 0 {
		fmt.Fprintf(Output, "%s %d\n", Bold("GPU Layers:"), p.GPULayers)
	}
	if p.Threads > 0 {
		fmt.Fprintf(Output, "%s %d\n", Bold("Threads:"), p.Threads)
	}

	fmt.Fprintf(Output, "%s %s\n", Bold("Endpoint:"), Blue(fmt.Sprintf("%s:%d", p.Host, p.Port)))

	if len(p.ExtraArgs) > 0 {
		fmt.Fprintf(Output, "%s %v\n", Bold("Extra Args:"), p.ExtraArgs)
	}
}

// PrintModelDetails prints model metadata in a formatted style.
func PrintModelDetails(m ModelDetails) {
	fmt.Fprintf(Output, "%s %s\n", Bold("Repository:"), Cyan(m.Repo))
	fmt.Fprintf(Output, "%s %s\n", Bold("Quantization:"), Yellow(m.Quant))
	fmt.Fprintf(Output, "%s %s\n", Bold("Filename:"), m.Filename)
	fmt.Fprintf(Output, "%s %s\n", Bold("Path:"), Blue(m.Path))
	fmt.Fprintf(Output, "%s %s\n", Bold("Size:"), m.Size)
	fmt.Fprintf(Output, "%s %s\n", Bold("Downloaded:"), m.DownloadedAt)
	fmt.Fprintf(Output, "%s %s\n", Bold("Status:"), Green("✓ Downloaded"))
}
