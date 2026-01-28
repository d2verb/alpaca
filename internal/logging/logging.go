// Package logging provides logging configuration with file rotation.
package logging

import (
	"io"
	"log/slog"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds log file configuration.
type Config struct {
	Path       string // Log file path
	MaxSizeMB  int    // Max size in MB before rotation
	MaxBackups int    // Number of old files to keep
	MaxAgeDays int    // Max age in days
	Compress   bool   // Compress old files
}

// DefaultConfig returns sensible defaults for log rotation.
func DefaultConfig(path string) Config {
	return Config{
		Path:       path,
		MaxSizeMB:  50,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   true,
	}
}

// NewRotatingWriter creates a log writer with rotation support.
func NewRotatingWriter(cfg Config) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
	}
}

// NewLogger creates a structured logger that writes to the given writer.
func NewLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
