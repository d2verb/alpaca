package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/d2verb/alpaca/internal/receipt"
	"github.com/d2verb/alpaca/internal/ui"
)

func TestUpgradeCmd_HandleFingerprintMismatch(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	cmd := &UpgradeCmd{Force: false}
	r := &receipt.Receipt{
		InstallSource: receipt.SourceBrew,
		BinaryPath:    "/usr/local/bin/alpaca",
	}

	// Act
	err := cmd.handleFingerprintMismatch(r)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()

	// Verify warning is displayed
	if !strings.Contains(output, "Binary does not match installation record") {
		t.Error("expected warning about fingerprint mismatch")
	}

	// Verify original install info is shown
	if !strings.Contains(output, "Original method: brew") {
		t.Errorf("expected original method to be shown, got: %s", output)
	}
	if !strings.Contains(output, "Original path:   /usr/local/bin/alpaca") {
		t.Errorf("expected original path to be shown, got: %s", output)
	}

	// Verify --force guidance is provided
	if !strings.Contains(output, "alpaca upgrade --force") {
		t.Error("expected --force guidance")
	}
}

func TestUpgradeCmd_HandleBrewInstall(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	cmd := &UpgradeCmd{}

	// Act
	err := cmd.handleBrewInstall()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Installed via Homebrew") {
		t.Error("expected Homebrew install message")
	}
	if !strings.Contains(output, "brew upgrade alpaca") {
		t.Error("expected brew upgrade command")
	}
}

func TestUpgradeCmd_HandleAptInstall(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	cmd := &UpgradeCmd{}

	// Act
	err := cmd.handleAptInstall()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Installed via apt") {
		t.Error("expected apt install message")
	}
	if !strings.Contains(output, "sudo apt update && sudo apt upgrade alpaca") {
		t.Error("expected apt upgrade command")
	}
}

func TestUpgradeCmd_HandleGoInstall(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	cmd := &UpgradeCmd{}

	// Act
	err := cmd.handleGoInstall()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Installed via go install") {
		t.Error("expected go install message")
	}
	if !strings.Contains(output, "go install github.com/d2verb/alpaca/cmd/alpaca@latest") {
		t.Error("expected go install command")
	}
}

func TestUpgradeCmd_HandleUnknownInstall_WithoutForce(t *testing.T) {
	// Arrange
	var buf bytes.Buffer
	ui.Output = &buf
	defer func() { ui.Output = os.Stdout }()

	cmd := &UpgradeCmd{Force: false}

	// Act
	err := cmd.handleUnknownInstall()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "Could not determine how alpaca was installed") {
		t.Error("expected unknown install warning")
	}
	if !strings.Contains(output, "alpaca upgrade --force") {
		t.Error("expected --force guidance")
	}
}
