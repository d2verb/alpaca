package daemon

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/d2verb/alpaca/internal/metadata"
)

func TestNewDaemonStartsIdle(t *testing.T) {
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	if d.State() != StateIdle {
		t.Errorf("State() = %q, want %q", d.State(), StateIdle)
	}
	if d.CurrentPreset() != nil {
		t.Error("CurrentPreset() should be nil")
	}
}

func TestListPresetsViaInterface(t *testing.T) {
	presets := &stubPresetLoader{names: []string{"codellama", "mistral"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	names, err := d.ListPresets()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(names) != 2 {
		t.Fatalf("len(names) = %d, want 2", len(names))
	}
	if names[0] != "codellama" {
		t.Errorf("names[0] = %q, want %q", names[0], "codellama")
	}
	if names[1] != "mistral" {
		t.Errorf("names[1] = %q, want %q", names[1], "mistral")
	}
}

func TestListModelsViaInterface(t *testing.T) {
	entries := []metadata.ModelEntry{
		{Repo: "TheBloke/CodeLlama-7B-GGUF", Quant: "Q4_K_M", Size: 1024},
	}
	models := &stubModelManager{entries: entries}
	presets := &stubPresetLoader{}
	d := newTestDaemon(presets, models)

	infos, err := d.ListModels(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1", len(infos))
	}
	if infos[0].Repo != "TheBloke/CodeLlama-7B-GGUF" {
		t.Errorf("Repo = %q, want %q", infos[0].Repo, "TheBloke/CodeLlama-7B-GGUF")
	}
}

func TestStateIsLockFree(t *testing.T) {
	// This test verifies that State() and CurrentPreset() can be called
	// concurrently without blocking, even when Run() holds the mutex.
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	// Manually acquire the mutex to simulate Run() holding it
	d.mu.Lock()

	// State() and CurrentPreset() should return immediately (lock-free)
	done := make(chan struct{})
	go func() {
		_ = d.State()
		_ = d.CurrentPreset()
		close(done)
	}()

	// Wait with timeout - if State()/CurrentPreset() were blocked by the mutex,
	// this would timeout
	select {
	case <-done:
		// Success: State() and CurrentPreset() returned without blocking
	case <-time.After(100 * time.Millisecond):
		t.Fatal("State() or CurrentPreset() blocked on mutex - they should be lock-free")
	}

	d.mu.Unlock()
}

func TestConcurrentStateAccess(t *testing.T) {
	// Test that multiple goroutines can safely read state concurrently.
	// The race detector (-race flag) will catch any data races.
	presets := &stubPresetLoader{names: []string{"test"}}
	models := &stubModelManager{}
	d := newTestDaemon(presets, models)

	const numReaders = 100
	var wg sync.WaitGroup
	wg.Add(numReaders)

	for range numReaders {
		go func() {
			defer wg.Done()
			for range 1000 {
				_ = d.State()
				_ = d.CurrentPreset()
			}
		}()
	}

	wg.Wait()
}
