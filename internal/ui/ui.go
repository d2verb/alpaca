// Package ui provides formatted output utilities for the CLI.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

// Semantic color functions - use these for new code.
var (
	// Primary: Identifiers, names, and primary data
	Primary = color.New(color.FgCyan, color.Bold).SprintFunc()

	// Secondary: Supplementary data (quant, type, etc.)
	Secondary = color.New(color.FgMagenta).SprintFunc()

	// Link: Paths, URLs (clickable impression)
	Link = color.New(color.FgBlue, color.Underline).SprintFunc()

	// Status colors
	Success = color.New(color.FgGreen).SprintFunc()
	Error   = color.New(color.FgRed).SprintFunc()
	Warning = color.New(color.FgYellow).SprintFunc()
	Info    = color.New(color.FgCyan).SprintFunc()

	// Muted: Supplementary info (size, timestamps, etc.) - using normal color for readability
	Muted = func(s string) string { return s }

	// Emphasis (headers, labels)
	Heading = color.New(color.FgWhite, color.Bold).SprintFunc()
	Label   = func(s string) string { return s } // Normal color for readability
)

// Legacy color functions - kept for backward compatibility.
// Prefer semantic colors (Primary, Secondary, etc.) for new code.
var (
	Green  = color.New(color.FgGreen).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Yellow = color.New(color.FgYellow).SprintFunc()
	Blue   = color.New(color.FgBlue).SprintFunc()
)

// Output is the destination for UI output.
// Defaults to os.Stdout but can be overridden for testing.
var Output io.Writer = os.Stdout

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
	fmt.Fprintf(Output, "🚀 %s\n", Heading("Status"))

	PrintKeyValue("State", StatusBadge(state))
	if preset != "" {
		label, formatted := formatPresetOrModel(preset)
		PrintKeyValue(label, formatted)
	}
	if endpoint != "" {
		PrintKeyValue("Endpoint", Link(endpoint))
	}
	PrintKeyValue("Logs", logPath)
}

// formatPresetOrModel formats a preset or model identifier and returns the label and formatted string.
// It handles h:, p:, and f: prefixes, as well as identifiers without prefixes.
func formatPresetOrModel(id string) (label, formatted string) {
	// Check if identifier has a valid prefix
	if len(id) < 2 || id[1] != ':' {
		// No prefix - treat as preset name
		return "Preset", fmt.Sprintf("%s%s", Primary("p:"), Primary(id))
	}

	prefix := id[0]
	rest := id[2:]

	switch prefix {
	case 'h':
		// HuggingFace model: h:org/repo:quant
		lastColon := strings.LastIndex(rest, ":")
		if lastColon != -1 {
			repo := rest[:lastColon]
			quant := rest[lastColon+1:]
			return "Model", fmt.Sprintf("%s%s:%s", Primary("h:"), Primary(repo), Secondary(quant))
		}
		return "Model", fmt.Sprintf("%s%s", Primary("h:"), Primary(rest))

	case 'f':
		// File path: f:/path/to/file
		return "Model", fmt.Sprintf("%s%s", Primary("f:"), Link(rest))

	case 'p':
		// Preset: p:name
		return "Preset", fmt.Sprintf("%s%s", Primary("p:"), Primary(rest))

	default:
		// Unknown prefix - treat as preset name
		return "Preset", fmt.Sprintf("%s%s", Primary("p:"), Primary(id))
	}
}

// PrintModelList prints a list of downloaded models with formatting.
func PrintModelList(models []ModelInfo) {
	PrintSectionHeader("🤖", "Models")
	if len(models) == 0 {
		fmt.Fprintf(Output, "  %s\n", Muted("(none)"))
		return
	}

	for _, m := range models {
		// Display in full h:repo:quant format (matches command input)
		fmt.Fprintf(Output, "  %s%s:%s\n",
			Primary("h:"),
			Primary(m.Repo),
			Secondary(m.Quant),
		)
		// Compact metadata on second line
		fmt.Fprintf(Output, "    %s · Downloaded %s\n",
			m.SizeString,
			m.DownloadedAt,
		)
	}
}

// ModelInfo represents a downloaded model for display.
type ModelInfo struct {
	Repo         string
	Quant        string
	SizeString   string
	DownloadedAt string
}

// PrintPresetList prints a list of available presets with formatting.
func PrintPresetList(presets []string) {
	PrintSectionHeader("📦", "Presets")
	if len(presets) == 0 {
		fmt.Fprintf(Output, "  %s\n", Muted("(none)"))
		return
	}

	for _, name := range presets {
		// Display with p: prefix (matches command input)
		fmt.Fprintf(Output, "  %s%s\n", Primary("p:"), Primary(name))
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

// PrintInfo prints an info message with info icon.
func PrintInfo(message string) {
	fmt.Fprintf(Output, "%s %s\n", Info("ℹ"), message)
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
	// Display with p: prefix
	identifier := fmt.Sprintf("%s%s", Primary("p:"), Primary(p.Name))
	PrintDetailHeader("📦", "Preset", identifier)

	PrintKeyValue("Model", Link(p.Model))
	if p.ContextSize > 0 {
		PrintKeyValue("Context Size", fmt.Sprintf("%d", p.ContextSize))
	}
	if p.GPULayers != 0 {
		PrintKeyValue("GPU Layers", fmt.Sprintf("%d", p.GPULayers))
	}
	if p.Threads > 0 {
		PrintKeyValue("Threads", fmt.Sprintf("%d", p.Threads))
	}
	PrintKeyValue("Endpoint", Link(fmt.Sprintf("%s:%d", p.Host, p.Port)))
	if len(p.ExtraArgs) > 0 {
		// Join args into a single string
		argsStr := ""
		for i, arg := range p.ExtraArgs {
			if i > 0 {
				argsStr += " "
			}
			argsStr += arg
		}
		PrintKeyValue("Extra Args", argsStr)
	}
}

// PrintModelDetails prints model metadata in a formatted style.
func PrintModelDetails(m ModelDetails) {
	// Display in full h:repo:quant format
	identifier := fmt.Sprintf("%s%s:%s",
		Primary("h:"),
		Primary(m.Repo),
		Secondary(m.Quant),
	)
	PrintDetailHeader("🤖", "Model", identifier)

	PrintKeyValue("Filename", m.Filename)
	PrintKeyValue("Size", m.Size)
	PrintKeyValue("Downloaded", m.DownloadedAt)
	PrintKeyValue("Path", Link(m.Path))
	PrintKeyValue("Status", Success("✓ Ready"))
}

// PrintSectionHeader prints a section header with divider for list outputs.
func PrintSectionHeader(icon, title string) {
	fmt.Fprintf(Output, "%s %s\n", icon, Heading(title))
	// Divider length: icon (2 chars including icon + 2 chars including space,) + title length
	dividerLen := len(title) + 4
	fmt.Fprintln(Output, Muted(repeatString("─", dividerLen)))
}

// PrintDetailHeader prints a header for detail views (no divider).
func PrintDetailHeader(icon, title, identifier string) {
	fmt.Fprintf(Output, "%s %s: %s\n", icon, Heading(title), identifier)
}

// PrintKeyValue prints a key-value pair with aligned formatting.
func PrintKeyValue(key, value string) {
	fmt.Fprintf(Output, "  %-14s %s\n", key, value)
}

// repeatString repeats a string n times.
func repeatString(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
