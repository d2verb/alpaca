package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// MmprojEntry represents metadata for a multimodal projector file.
type MmprojEntry struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
}

// ModelEntry represents metadata for a downloaded model.
type ModelEntry struct {
	Repo         string       `json:"repo"`
	Quant        string       `json:"quant"`
	Filename     string       `json:"filename"`
	Size         int64        `json:"size"`
	Mmproj       *MmprojEntry `json:"mmproj,omitempty"`
	DownloadedAt time.Time    `json:"downloaded_at"`
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

	// Atomic write: temp file + rename to prevent corruption on crash
	tmp, err := os.CreateTemp(dir, ".metadata-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, m.filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename metadata file: %w", err)
	}

	return nil
}

// Add adds or updates a model entry.
// If an entry with the same repo+quant exists, it's replaced.
func (m *Manager) Add(entry ModelEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing entry if present
	m.data.Models = slices.DeleteFunc(m.data.Models, func(e ModelEntry) bool {
		return e.Repo == entry.Repo && e.Quant == entry.Quant
	})

	// Add new entry
	m.data.Models = append(m.data.Models, entry)
	return nil
}

// Remove removes a model entry.
// Returns nil if the entry doesn't exist.
func (m *Manager) Remove(repo, quant string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data.Models = slices.DeleteFunc(m.data.Models, func(e ModelEntry) bool {
		return e.Repo == repo && e.Quant == quant
	})

	return nil
}

// Find looks up a model entry.
// Returns a copy of the entry; mutations do not affect the underlying data.
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
	return slices.Clone(m.data.Models)
}

// MmprojReferenceCount returns the number of model entries that reference
// the given mmproj filename. This is used for reference counting when
// deleting or cleaning up mmproj files.
func (m *Manager) MmprojReferenceCount(filename string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, e := range m.data.Models {
		if e.Mmproj != nil && e.Mmproj.Filename == filename {
			count++
		}
	}
	return count
}

// GetFilePath resolves repo:quant to the actual file path.
// Returns an error if the model is not found in metadata.
func (m *Manager) GetFilePath(modelsDir, repo, quant string) (string, error) {
	entry := m.Find(repo, quant)
	if entry == nil {
		return "", &NotFoundError{Repo: repo, Quant: quant}
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
