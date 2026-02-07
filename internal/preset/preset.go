// Package preset handles preset management.
package preset

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// namePattern validates preset names: alphanumeric, underscore, hyphen only.
var namePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// invalidCharsPattern matches characters that are not alphanumeric, underscore, or hyphen.
var invalidCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

const (
	// DefaultPort is the default port for llama-server.
	DefaultPort = 8080
	// DefaultHost is the default host for llama-server.
	DefaultHost = "127.0.0.1"
	// DefaultContextSize is the default context size for llama-server.
	DefaultContextSize = 4096
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

// SanitizeName converts an arbitrary string to a valid preset name.
// Invalid characters are replaced with hyphens, consecutive hyphens are
// collapsed to a single hyphen, and leading/trailing hyphens are trimmed.
func SanitizeName(name string) string {
	// Replace invalid characters with hyphens
	result := invalidCharsPattern.ReplaceAllString(name, "-")

	// Collapse consecutive hyphens to a single hyphen
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading and trailing hyphens
	result = strings.Trim(result, "-")

	return result
}

// Preset represents a model + argument combination.
type Preset struct {
	Name        string   `yaml:"name"` // Required, used as identifier
	Model       string   `yaml:"model"`
	DraftModel  string   `yaml:"draft_model,omitempty"`
	ContextSize int      `yaml:"context_size,omitempty"`
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

	if p.DraftModel != "" {
		draftModelPath := strings.TrimPrefix(p.DraftModel, "f:")
		args = append(args, "--model-draft", draftModelPath)
	}

	// Always include context size with default
	args = append(args, "--ctx-size", strconv.Itoa(p.GetContextSize()))

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
