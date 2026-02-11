package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/d2verb/alpaca/internal/receipt"
	"github.com/d2verb/alpaca/internal/selfupdate"
	"github.com/d2verb/alpaca/internal/ui"
)

type UpgradeCmd struct {
	Force bool `help:"Force upgrade even if installation source is unknown or mismatched" short:"f"`
	Check bool `help:"Check for updates without installing" short:"c"`
}

func (c *UpgradeCmd) Run() error {
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// Detect install source by path first
	source := receipt.DetectInstallSource(currentBinary)

	// For package managers, use path-based detection directly
	switch source {
	case receipt.SourceBrew:
		return c.handleBrewInstall()
	case receipt.SourceApt:
		return c.handleAptInstall()
	case receipt.SourceGo:
		return c.handleGoInstall()
	case receipt.SourceUnknown:
		return c.handleUnknownInstall()
	}

	// For script installs, check receipt for fingerprint verification
	r, err := receipt.Load()
	if err != nil {
		if errors.Is(err, receipt.ErrNotFound) {
			return c.handleUnknownInstall()
		}
		return fmt.Errorf("load receipt: %w", err)
	}

	// Verify the receipt matches current binary
	verifyResult, err := r.Verify(currentBinary)
	if err != nil {
		return fmt.Errorf("verify receipt: %w", err)
	}

	// Handle verification results
	if verifyResult.FingerprintMismatch {
		if !c.Force {
			return c.handleFingerprintMismatch(r)
		}
		ui.PrintWarning("Proceeding despite fingerprint mismatch (--force)")
	} else if verifyResult.PathMismatch {
		// Fingerprint matches but path differs (e.g., symlink)
		ui.PrintInfo(fmt.Sprintf("Binary path differs (receipt: %s)", r.BinaryPath))
		ui.PrintInfo("Fingerprint matches, proceeding...")
	}

	return c.handleScriptInstall(currentBinary, r)
}

func (c *UpgradeCmd) handleScriptInstall(currentBinary string, r *receipt.Receipt) error {
	ui.PrintInfo("Checking for updates...")

	updater := selfupdate.New(version)

	// Check for new version
	latest, hasUpdate, err := updater.CheckUpdate(context.Background())
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}

	fmt.Fprintln(ui.Output)
	fmt.Fprintf(ui.Output, "  Current: %s\n", version)
	fmt.Fprintf(ui.Output, "  Latest:  %s\n", latest)
	fmt.Fprintln(ui.Output)

	if !hasUpdate {
		ui.PrintSuccess("Already up to date")
		return nil
	}

	if c.Check {
		ui.PrintInfo("Update available. Run: alpaca upgrade")
		return nil
	}

	// Perform the upgrade
	ui.PrintInfo("Downloading...")

	if err := updater.Update(context.Background(), currentBinary); err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Update the receipt
	if err := r.UpdateFingerprint(currentBinary); err != nil {
		return fmt.Errorf("update receipt: %w", err)
	}
	if err := r.Save(); err != nil {
		return fmt.Errorf("save receipt: %w", err)
	}

	ui.PrintSuccess(fmt.Sprintf("Upgraded to %s", latest))
	return nil
}

func (c *UpgradeCmd) handleBrewInstall() error {
	ui.PrintInfo("Installed via Homebrew.")
	fmt.Fprintln(ui.Output, "To upgrade, run:")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "    brew upgrade alpaca")
	fmt.Fprintln(ui.Output)
	return nil
}

func (c *UpgradeCmd) handleAptInstall() error {
	ui.PrintInfo("Installed via apt.")
	fmt.Fprintln(ui.Output, "To upgrade, run:")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "    sudo apt update && sudo apt upgrade alpaca")
	fmt.Fprintln(ui.Output)
	return nil
}

func (c *UpgradeCmd) handleGoInstall() error {
	ui.PrintInfo("Installed via go install.")
	fmt.Fprintln(ui.Output, "To upgrade, run:")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "    go install github.com/d2verb/alpaca/cmd/alpaca@latest")
	fmt.Fprintln(ui.Output)
	return nil
}

func (c *UpgradeCmd) handleUnknownInstall() error {
	ui.PrintWarning("Could not determine how alpaca was installed.")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "If you installed via the install script, force an upgrade:")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "    alpaca upgrade --force")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "Otherwise, use your original installation method.")

	if c.Force {
		fmt.Fprintln(ui.Output)
		ui.PrintInfo("Attempting upgrade with --force...")

		currentBinary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("get executable path: %w", err)
		}

		// Create a proper temporary receipt for the forced upgrade
		r := &receipt.Receipt{
			SchemaVersion: receipt.SchemaVersion,
			InstallSource: receipt.SourceScript,
			BinaryPath:    currentBinary,
		}
		return c.handleScriptInstall(currentBinary, r)
	}

	return nil
}

func (c *UpgradeCmd) handleFingerprintMismatch(r *receipt.Receipt) error {
	ui.PrintWarning("Binary does not match installation record.")
	fmt.Fprintln(ui.Output, "This may happen if alpaca was reinstalled via a different method.")
	fmt.Fprintln(ui.Output)
	fmt.Fprintf(ui.Output, "  Original method: %s\n", r.InstallSource)
	fmt.Fprintf(ui.Output, "  Original path:   %s\n", r.BinaryPath)
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "To proceed anyway:")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "    alpaca upgrade --force")
	fmt.Fprintln(ui.Output)
	fmt.Fprintln(ui.Output, "Or use your current package manager's upgrade command.")

	return nil
}
