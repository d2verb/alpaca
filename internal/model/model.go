// Package model handles model file operations.
package model

import (
	"context"
	"fmt"
	"os"

	"github.com/d2verb/alpaca/internal/metadata"
)

// Manager handles model file operations.
type Manager struct {
	modelsDir string
	metadata  *metadata.Manager
}

// NewManager creates a new model manager.
func NewManager(modelsDir string) *Manager {
	return &Manager{
		modelsDir: modelsDir,
		metadata:  metadata.NewManager(modelsDir),
	}
}

// List returns all downloaded models from metadata.
func (m *Manager) List(ctx context.Context) ([]metadata.ModelEntry, error) {
	if err := m.metadata.Load(ctx); err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}
	return m.metadata.List(), nil
}

// Remove deletes a model file and its metadata entry.
func (m *Manager) Remove(ctx context.Context, repo, quant string) error {
	if err := m.metadata.Load(ctx); err != nil {
		return fmt.Errorf("load metadata: %w", err)
	}

	// Get file path from metadata
	filePath, err := m.metadata.GetFilePath(m.modelsDir, repo, quant)
	if err != nil {
		return err
	}

	// Remove file
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}

	// Remove metadata entry
	if err := m.metadata.Remove(repo, quant); err != nil {
		return fmt.Errorf("remove metadata: %w", err)
	}
	if err := m.metadata.Save(ctx); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}

	return nil
}

// Exists checks if a model is downloaded.
func (m *Manager) Exists(ctx context.Context, repo, quant string) (bool, error) {
	if err := m.metadata.Load(ctx); err != nil {
		return false, fmt.Errorf("load metadata: %w", err)
	}

	entry := m.metadata.Find(repo, quant)
	return entry != nil, nil
}

// GetFilePath resolves repo:quant to the actual file path.
func (m *Manager) GetFilePath(ctx context.Context, repo, quant string) (string, error) {
	if err := m.metadata.Load(ctx); err != nil {
		return "", fmt.Errorf("load metadata: %w", err)
	}

	return m.metadata.GetFilePath(m.modelsDir, repo, quant)
}

// GetDetails returns detailed information about a model.
func (m *Manager) GetDetails(ctx context.Context, repo, quant string) (*metadata.ModelEntry, error) {
	if err := m.metadata.Load(ctx); err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}

	entry := m.metadata.Find(repo, quant)
	if entry == nil {
		return nil, fmt.Errorf("model not found")
	}

	return entry, nil
}
