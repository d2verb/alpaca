package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/d2verb/alpaca/internal/identifier"
	"github.com/d2verb/alpaca/internal/pathutil"
	"github.com/d2verb/alpaca/internal/protocol"
	"github.com/d2verb/alpaca/internal/ui"
)

// LocalPresetFile is the filename for local presets.
const LocalPresetFile = ".alpaca.yaml"

type LoadCmd struct {
	Identifier string `arg:"" optional:"" help:"Identifier (p:preset, h:org/repo:quant, f:/path/to/file, or f:*.yaml)" predictor:"load-identifier"`
}

func (c *LoadCmd) Run() error {
	cl, err := newClient()
	if err != nil {
		return err
	}

	// Resolve identifier (handles empty input → .alpaca.yaml)
	idStr, err := c.resolveIdentifier()
	if err != nil {
		return err
	}

	// Parse and normalize identifier
	id, err := identifier.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid identifier: %w", err)
	}

	// Prepare load request (normalize paths, get display name)
	req, err := c.prepare(id)
	if err != nil {
		return err
	}

	// Send to daemon (daemon handles auto-pull)
	ui.PrintInfo(fmt.Sprintf("Loading %s...", req.displayName))
	resp, err := cl.Load(req.identifier, true)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.ECONNREFUSED) {
			return errDaemonNotRunning()
		}
		return fmt.Errorf("load model: %w", err)
	}

	if resp.Status == "error" {
		return handleLoadError(resp.ErrorCode, resp.Error, id)
	}

	endpoint, _ := resp.Data["endpoint"].(string)
	ui.PrintSuccess(fmt.Sprintf("Model ready at %s", ui.FormatEndpoint(endpoint)))
	return nil
}

// resolveIdentifier resolves the identifier from input or defaults to .alpaca.yaml.
func (c *LoadCmd) resolveIdentifier() (string, error) {
	if c.Identifier != "" {
		return c.Identifier, nil
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

// loadRequest holds the prepared load request data.
type loadRequest struct {
	identifier  string
	displayName string
}

// prepare normalizes paths and determines display name.
func (c *LoadCmd) prepare(id *identifier.Identifier) (*loadRequest, error) {
	req := &loadRequest{
		identifier:  id.Raw,
		displayName: id.Raw,
	}

	// Handle file path types (both preset and model)
	if id.Type == identifier.TypePresetFilePath || id.Type == identifier.TypeModelFilePath {
		absID, err := toAbsFileID(id.FilePath)
		if err != nil {
			return nil, err
		}
		req.identifier = absID
		req.displayName = id.FilePath
	}

	return req, nil
}

// toAbsFileID converts a file path to absolute f: identifier.
func toAbsFileID(path string) (string, error) {
	// Resolve tilde expansion first
	resolved, err := pathutil.ResolvePath(path, "")
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}
	// Then make absolute (for relative paths like ./preset.yaml)
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("make absolute path: %w", err)
	}
	return "f:" + absPath, nil
}

// handleLoadError converts daemon error codes into user-friendly errors.
func handleLoadError(code, message string, id *identifier.Identifier) error {
	switch code {
	case protocol.ErrCodePresetNotFound:
		return errPresetNotFound(id.PresetName)

	case protocol.ErrCodeModelNotFound:
		if id.Type == identifier.TypePresetName {
			return fmt.Errorf("model in preset '%s' not downloaded\nRun: alpaca pull <model>", id.PresetName)
		}
		return errModelNotFound(id.Raw)

	case protocol.ErrCodeServerFailed:
		return fmt.Errorf("failed to start server: %s", message)

	default:
		return fmt.Errorf("%s", message)
	}
}
