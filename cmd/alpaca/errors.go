package main

import "fmt"

// Exit codes for CLI commands.
const (
	exitSuccess          = 0
	exitError            = 1
	exitDaemonNotRunning = 2
	exitPresetNotFound   = 3
	exitModelNotFound    = 4
	exitDownloadFailed   = 5
)

// ExitError represents an error that should cause the process to exit with a specific code.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string { return e.Message }

func errDaemonNotRunning() *ExitError {
	return &ExitError{
		Code:    exitDaemonNotRunning,
		Message: "Daemon is not running.\nRun: alpaca start",
	}
}

func errPresetNotFound(name string) *ExitError {
	return &ExitError{
		Code:    exitPresetNotFound,
		Message: fmt.Sprintf("Preset '%s' not found.", name),
	}
}

func errModelNotFound(id string) *ExitError {
	return &ExitError{
		Code:    exitModelNotFound,
		Message: fmt.Sprintf("Model '%s' not found.", id),
	}
}

func errDownloadFailed() *ExitError {
	return &ExitError{
		Code:    exitDownloadFailed,
		Message: "",
	}
}
