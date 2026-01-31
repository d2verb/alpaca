package main

import (
	"errors"
	"testing"
)

func TestExitErrorImplementsError(t *testing.T) {
	err := &ExitError{Code: 1, Message: "something failed"}

	got := err.Error()
	want := "something failed"

	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestExitErrorUnwrapWithErrorsAs(t *testing.T) {
	var wrapped error = &ExitError{Code: 2, Message: "daemon not running"}

	var exitErr *ExitError
	if !errors.As(wrapped, &exitErr) {
		t.Fatal("errors.As did not match ExitError")
	}

	if exitErr.Code != 2 {
		t.Errorf("Code = %d, want 2", exitErr.Code)
	}
}

func TestErrDaemonNotRunning(t *testing.T) {
	err := errDaemonNotRunning()

	if err.Code != exitDaemonNotRunning {
		t.Errorf("Code = %d, want %d", err.Code, exitDaemonNotRunning)
	}
	if err.Message == "" {
		t.Error("Message should not be empty")
	}
}

func TestErrPresetNotFound(t *testing.T) {
	err := errPresetNotFound("codellama")

	if err.Code != exitPresetNotFound {
		t.Errorf("Code = %d, want %d", err.Code, exitPresetNotFound)
	}
	if err.Message != "Preset 'codellama' not found." {
		t.Errorf("Message = %q, want %q", err.Message, "Preset 'codellama' not found.")
	}
}

func TestErrModelNotFound(t *testing.T) {
	err := errModelNotFound("TheBloke/CodeLlama:Q4_K_M")

	if err.Code != exitModelNotFound {
		t.Errorf("Code = %d, want %d", err.Code, exitModelNotFound)
	}
	if err.Message != "Model 'TheBloke/CodeLlama:Q4_K_M' not found." {
		t.Errorf("Message = %q, want %q", err.Message, "Model 'TheBloke/CodeLlama:Q4_K_M' not found.")
	}
}

func TestErrDownloadFailed(t *testing.T) {
	err := errDownloadFailed()

	if err.Code != exitDownloadFailed {
		t.Errorf("Code = %d, want %d", err.Code, exitDownloadFailed)
	}
	if err.Message != "" {
		t.Errorf("Message = %q, want empty", err.Message)
	}
}

func TestErrServerNotRunning(t *testing.T) {
	err := errServerNotRunning()

	if err.Code != exitError {
		t.Errorf("Code = %d, want %d", err.Code, exitError)
	}
	if err.Message == "" {
		t.Error("Message should not be empty")
	}
}
