package daemon

import (
	"context"
	"io"

	"github.com/d2verb/alpaca/internal/llama"
	"github.com/d2verb/alpaca/internal/metadata"
	"github.com/d2verb/alpaca/internal/preset"
)

type stubPresetLoader struct {
	presets map[string]*preset.Preset
	names   []string
	listErr error
}

func (s *stubPresetLoader) Load(name string) (*preset.Preset, error) {
	p, ok := s.presets[name]
	if !ok {
		return nil, &preset.NotFoundError{Name: name}
	}
	return p, nil
}

func (s *stubPresetLoader) List() ([]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.names, nil
}

type stubModelManager struct {
	entries  []metadata.ModelEntry
	filePath string
	exists   bool
	err      error
}

func (s *stubModelManager) List(ctx context.Context) ([]metadata.ModelEntry, error) {
	return s.entries, s.err
}

func (s *stubModelManager) GetFilePath(ctx context.Context, repo, quant string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if !s.exists {
		return "", &metadata.NotFoundError{Repo: repo, Quant: quant}
	}
	return s.filePath, nil
}

func (s *stubModelManager) GetDetails(ctx context.Context, repo, quant string) (*metadata.ModelEntry, error) {
	if s.err != nil {
		return nil, s.err
	}
	for _, e := range s.entries {
		if e.Repo == repo && e.Quant == quant {
			return &e, nil
		}
	}
	return nil, &metadata.NotFoundError{Repo: repo, Quant: quant}
}

func newTestDaemon(presets presetLoader, models modelManager) *Daemon {
	return New(presets, models, "", io.Discard, io.Discard)
}

func newTestDaemonWithConfigPath(presets presetLoader, models modelManager, configPath string) *Daemon {
	return New(presets, models, configPath, io.Discard, io.Discard)
}

// mockProcess is a mock implementation of llamaProcess for testing.
type mockProcess struct {
	startErr     error
	stopErr      error
	startCalled  bool
	stopCalled   bool
	logWriter    io.Writer
	receivedArgs []string
	doneCh       chan struct{}
	exitError    error
}

func (m *mockProcess) Start(args []string) error {
	m.startCalled = true
	m.receivedArgs = args
	if m.startErr != nil {
		return &llama.ProcessError{Op: llama.ProcessOpStart, Err: m.startErr}
	}
	return nil
}

func (m *mockProcess) Stop(ctx context.Context) error {
	m.stopCalled = true
	return m.stopErr
}

func (m *mockProcess) SetLogWriter(w io.Writer) {
	m.logWriter = w
}

// Done returns doneCh. When doneCh is nil (default), the returned nil channel
// blocks forever in select, simulating a process that never exits.
func (m *mockProcess) Done() <-chan struct{} {
	return m.doneCh
}

func (m *mockProcess) ExitErr() error {
	return m.exitError
}

// mockHealthChecker returns a health checker function that can be configured to succeed or fail.
func mockHealthChecker(err error) healthChecker {
	return func(ctx context.Context, endpoint string) error {
		return err
	}
}

// mapModelManager resolves different models based on repo+quant key.
type mapModelManager struct {
	paths   map[string]string               // key: "repo:quant", value: file path
	entries map[string]*metadata.ModelEntry // key: "repo:quant", optional detailed entries
}

func (m *mapModelManager) List(ctx context.Context) ([]metadata.ModelEntry, error) {
	return nil, nil
}

func (m *mapModelManager) GetFilePath(ctx context.Context, repo, quant string) (string, error) {
	key := repo + ":" + quant
	path, ok := m.paths[key]
	if !ok {
		return "", &metadata.NotFoundError{Repo: repo, Quant: quant}
	}
	return path, nil
}

func (m *mapModelManager) GetDetails(ctx context.Context, repo, quant string) (*metadata.ModelEntry, error) {
	key := repo + ":" + quant
	if m.entries != nil {
		if entry, ok := m.entries[key]; ok {
			return entry, nil
		}
	}
	// Fall back to a basic entry if the path exists
	if _, ok := m.paths[key]; ok {
		return &metadata.ModelEntry{Repo: repo, Quant: quant}, nil
	}
	return nil, &metadata.NotFoundError{Repo: repo, Quant: quant}
}
