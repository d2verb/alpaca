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

// ExitKind represents the kind of exit error for UI formatting.
type ExitKind int

const (
	ExitKindError ExitKind = iota
	ExitKindInfo
)

// ExitError represents an error that should cause the process to exit with a specific code.
type ExitError struct {
	Code    int
	Kind    ExitKind
	Message string
}

func (e *ExitError) Error() string { return e.Message }

func errDaemonNotRunning() *ExitError {
	return &ExitError{
		Code:    exitDaemonNotRunning,
		Kind:    ExitKindInfo,
		Message: "Daemon is not running.\nRun: alpaca start",
	}
}

func errPresetNotFound(name string) *ExitError {
	return &ExitError{
		Code:    exitPresetNotFound,
		Kind:    ExitKindError,
		Message: fmt.Sprintf("Preset '%s' not found.", name),
	}
}

func errModelNotFound(id string) *ExitError {
	return &ExitError{
		Code:    exitModelNotFound,
		Kind:    ExitKindError,
		Message: fmt.Sprintf("Model '%s' not found.", id),
	}
}

func errDownloadFailed() *ExitError {
	return &ExitError{
		Code:    exitDownloadFailed,
		Kind:    ExitKindError,
		Message: "",
	}
}

func errServerNotRunning() *ExitError {
	return &ExitError{
		Code:    exitError,
		Kind:    ExitKindInfo,
		Message: "Server is not running.\nRun: alpaca load <preset>",
	}
}
