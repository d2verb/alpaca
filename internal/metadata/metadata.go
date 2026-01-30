package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ModelEntry represents metadata for a downloaded model.
type ModelEntry struct {
	Repo         string    `json:"repo"`
	Quant        string    `json:"quant"`
	Filename     string    `json:"filename"`
	Size         int64     `json:"size"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

// Metadata holds all model entries.
type Metadata struct {
	Models []ModelEntry `json:"models"`
}

// Manager handles metadata persistence.
type Manager struct {
	filePath string
	data     *Metadata
	mu       sync.Mutex
}

// NewManager creates a new metadata manager.
// The metadata file will be stored at modelsDir/.metadata.json
func NewManager(modelsDir string) *Manager {
	return &Manager{
		filePath: filepath.Join(modelsDir, ".metadata.json"),
		data:     &Metadata{Models: []ModelEntry{}},
	}
}

// Load reads metadata from disk.
// If the file doesn't exist, returns an empty metadata (not an error).
func (m *Manager) Load(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - treat as empty
			m.data = &Metadata{Models: []ModelEntry{}}
			return nil
		}
		return fmt.Errorf("read metadata file: %w", err)
	}

	if len(data) == 0 {
		// Empty file - treat as empty metadata
		m.data = &Metadata{Models: []ModelEntry{}}
		return nil
	}

	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parse metadata: %w", err)
	}

	m.data = &meta
	return nil
}

// Save writes metadata to disk.
func (m *Manager) Save(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(m.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("write metadata file: %w", err)
	}

	return nil
}

// Add adds or updates a model entry.
// If an entry with the same repo+quant exists, it's replaced.
func (m *Manager) Add(entry ModelEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing entry if present
	for i, e := range m.data.Models {
		if e.Repo == entry.Repo && e.Quant == entry.Quant {
			m.data.Models = append(m.data.Models[:i], m.data.Models[i+1:]...)
			break
		}
	}

	// Add new entry
	m.data.Models = append(m.data.Models, entry)
	return nil
}

// Remove removes a model entry.
// Returns nil if the entry doesn't exist.
func (m *Manager) Remove(repo, quant string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, e := range m.data.Models {
		if e.Repo == repo && e.Quant == quant {
			m.data.Models = append(m.data.Models[:i], m.data.Models[i+1:]...)
			return nil
		}
	}

	return nil
}

// Find looks up a model entry.
// Returns nil if not found.
func (m *Manager) Find(repo, quant string) *ModelEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, e := range m.data.Models {
		if e.Repo == repo && e.Quant == quant {
			return &e
		}
	}

	return nil
}

// List returns all model entries.
func (m *Manager) List() []ModelEntry {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent external mutation
	entries := make([]ModelEntry, len(m.data.Models))
	copy(entries, m.data.Models)
	return entries
}

// GetFilePath resolves repo:quant to the actual file path.
// Returns an error if the model is not found in metadata.
func (m *Manager) GetFilePath(modelsDir, repo, quant string) (string, error) {
	entry := m.Find(repo, quant)
	if entry == nil {
		return "", fmt.Errorf("model %s:%s not found in metadata", repo, quant)
	}

	filePath := filepath.Join(modelsDir, entry.Filename)

	// Verify file exists
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("model file not found: %s (try running 'alpaca ls' and cleanup)", filePath)
		}
		return "", fmt.Errorf("check model file: %w", err)
	}

	return filePath, nil
}
