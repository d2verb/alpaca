package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d2verb/alpaca/internal/client"
	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/preset"
	"github.com/d2verb/alpaca/internal/pull"
	"github.com/d2verb/alpaca/internal/ui"
)

// resolveLocalPreset resolves an identifier string from input or defaults to .alpaca.yaml.
// If id is non-empty, it is returned as-is. Otherwise, the current directory is checked
// for a local preset file and its path is returned as an f: identifier.
func resolveLocalPreset(id string) (string, error) {
	if id != "" {
		return id, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	presetPath := filepath.Join(cwd, LocalPresetFile)
	if _, err := os.Stat(presetPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no %s found in current directory\nRun: alpaca new --local", LocalPresetFile)
		}
		return "", fmt.Errorf("check preset file: %w", err)
	}

	return "f:" + presetPath, nil
}

// mapPresetError converts preset package errors to user-friendly errors.
func mapPresetError(err error, name string) error {
	if preset.IsNotFound(err) {
		return errPresetNotFound(name)
	}
	return err
}

// stdin is the input source for prompts. Can be replaced for testing.
var stdin = bufio.NewReader(os.Stdin)

// promptLine prompts the user for input and returns the trimmed response.
// If defaultVal is provided, it's shown in brackets and returned if input is empty.
func promptLine(label, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(ui.Output, "%s [%s]: ", label, defaultVal)
	} else {
		fmt.Fprintf(ui.Output, "%s: ", label)
	}
	input, err := stdin.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// promptConfirm prompts the user for a yes/no confirmation.
// Returns true only if user enters "y" or "Y".
func promptConfirm(message string) bool {
	ui.PrintConfirm(message)
	input, err := stdin.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(input)
	return input == "y" || input == "Y"
}

func getPaths() (*config.Paths, error) {
	paths, err := config.GetPaths()
	if err != nil {
		return nil, fmt.Errorf("get paths: %w", err)
	}
	return paths, nil
}

func newClient() (*client.Client, error) {
	paths, err := getPaths()
	if err != nil {
		return nil, err
	}
	return client.New(paths.Socket), nil
}

// pullModel downloads a model from HuggingFace.
func pullModel(repo, quant, modelsDir string) error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	if err := paths.EnsureDirectories(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	puller := pull.NewPuller(modelsDir)

	// Get file info first
	ui.PrintInfo("Fetching file list...")
	info, err := puller.GetFileInfo(context.Background(), repo, quant)
	if err != nil {
		return err
	}

	// Show download plan when multiple files
	if info.MmprojOriginalFilename != "" {
		ui.PrintInfo("Download plan (2 files):")
		nameWidth := len(info.Filename)
		if w := len(info.MmprojOriginalFilename); w > nameWidth {
			nameWidth = w
		}
		fmt.Fprintf(ui.Output, "  Model:  %-*s  (%s)\n", nameWidth, info.Filename, formatSize(info.Size))
		fmt.Fprintf(ui.Output, "  Mmproj: %-*s  (%s)\n", nameWidth, info.MmprojOriginalFilename, formatSize(info.MmprojSize))
	}

	// Set up progress reporting
	puller.SetProgressFunc(func(downloaded, total int64) {
		printProgress(downloaded, total)
	})

	// Set up file lifecycle callbacks
	puller.SetFileStartFunc(func(filename string, size int64, index, total int) {
		if total > 1 {
			ui.PrintInfo(fmt.Sprintf("[%d/%d] Downloading %s (%s)...", index, total, filename, formatSize(size)))
		} else {
			ui.PrintInfo(fmt.Sprintf("Downloading %s (%s)...", filename, formatSize(size)))
		}
	})
	puller.SetFileSavedFunc(func(savedPath string) {
		fmt.Fprintln(ui.Output) // End progress bar line
		ui.PrintSuccess(fmt.Sprintf("Saved to: %s", savedPath))
	})

	// Download
	result, err := puller.Pull(context.Background(), repo, quant)
	if err != nil {
		fmt.Fprintln(ui.Output) // End progress bar line
		return err
	}

	// Report mmproj failure
	if result.MmprojFailed {
		fmt.Fprintln(ui.Output) // End progress bar line
		ui.PrintWarning(fmt.Sprintf("mmproj download failed - vision unavailable. Run 'alpaca pull h:%s:%s' to retry.", repo, quant))
		return errDownloadFailed()
	}

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
