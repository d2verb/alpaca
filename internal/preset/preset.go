// Package preset handles preset management.
package preset

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/d2verb/alpaca/internal/identifier"
)

// namePattern validates preset names: alphanumeric, underscore, hyphen only.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	// DefaultPort is the default port for llama-server.
	DefaultPort = 8080
	// DefaultHost is the default host for llama-server.
	DefaultHost = "127.0.0.1"
	// DefaultContextSize is the default context size for llama-server.
	DefaultContextSize = 2048
	// DefaultGPULayers is the default number of GPU layers (-1 = all layers).
	DefaultGPULayers = -1
)

// ValidateName checks if a preset name is valid.
// Valid names contain only alphanumeric characters, underscores, and hyphens.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("name must contain only alphanumeric characters, underscores, and hyphens")
	}
	return nil
}

// Preset represents a model + argument combination.
type Preset struct {
	Name        string   `yaml:"name"` // Required, used as identifier
	Model       string   `yaml:"model"`
	ContextSize int      `yaml:"context_size,omitempty"`
	GPULayers   int      `yaml:"gpu_layers,omitempty"`
	Threads     int      `yaml:"threads,omitempty"`
	Port        int      `yaml:"port,omitempty"`
	Host        string   `yaml:"host,omitempty"`
	ExtraArgs   []string `yaml:"extra_args,omitempty"`
}

// GetPort returns the port, using default if not set.
func (p *Preset) GetPort() int {
	if p.Port > 0 {
		return p.Port
	}
	return DefaultPort
}

// GetHost returns the host, using default if not set.
func (p *Preset) GetHost() string {
	if p.Host != "" {
		return p.Host
	}
	return DefaultHost
}

// GetContextSize returns the context size, using default if not set.
func (p *Preset) GetContextSize() int {
	if p.ContextSize > 0 {
		return p.ContextSize
	}
	return DefaultContextSize
}

// GetGPULayers returns the GPU layers, using default if not set.
func (p *Preset) GetGPULayers() int {
	if p.GPULayers != 0 {
		return p.GPULayers
	}
	return DefaultGPULayers
}

// Endpoint returns the HTTP endpoint for this preset.
func (p *Preset) Endpoint() string {
	return fmt.Sprintf("http://%s:%d", p.GetHost(), p.GetPort())
}

// BuildArgs builds the command-line arguments for llama-server.
func (p *Preset) BuildArgs() []string {
	// Extract actual file path from f: prefix if present
	modelPath := strings.TrimPrefix(p.Model, "f:")

	args := []string{
		"-m", modelPath,
	}

	// Always include context size and GPU layers with defaults
	args = append(args, "--ctx-size", strconv.Itoa(p.GetContextSize()))
	args = append(args, "--n-gpu-layers", strconv.Itoa(p.GetGPULayers()))

	if p.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(p.Threads))
	}

	// Always include port and host with defaults
	args = append(args, "--port", strconv.Itoa(p.GetPort()))
	args = append(args, "--host", p.GetHost())

	// Expand extra args (split each element by whitespace)
	for _, arg := range p.ExtraArgs {
		args = append(args, strings.Fields(arg)...)
	}
	return args
}

// ModelResolver resolves HuggingFace model identifiers to file paths.
type ModelResolver interface {
	GetFilePath(ctx context.Context, repo, quant string) (string, error)
}

// ResolveModel resolves the model field in a preset if it's HuggingFace format.
// Returns a new preset with the resolved model path without mutating the original.
func ResolveModel(ctx context.Context, p *Preset, resolver ModelResolver) (*Preset, error) {
	id, err := identifier.Parse(p.Model)
	if err != nil {
		return nil, fmt.Errorf("invalid model field in preset: %w", err)
	}

	if id.Type == identifier.TypeHuggingFace {
		// Resolve HF identifier to file path
		modelPath, err := resolver.GetFilePath(ctx, id.Repo, id.Quant)
		if err != nil {
			return nil, fmt.Errorf("resolve model %s:%s: %w", id.Repo, id.Quant, err)
		}
		// Create new preset with resolved path (with f: prefix, don't mutate original)
		resolved := *p
		resolved.Model = "f:" + modelPath
		return &resolved, nil
	}

	// Already a file path (f:...), return as-is
	return p, nil
}
