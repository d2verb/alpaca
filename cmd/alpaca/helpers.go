package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/d2verb/alpaca/internal/client"
	"github.com/d2verb/alpaca/internal/config"
	"github.com/d2verb/alpaca/internal/pull"
)

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
