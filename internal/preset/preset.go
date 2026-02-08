// Package preset handles preset management.
package preset

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
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

// ModelEntry represents a single model in router mode.
type ModelEntry struct {
	Name          string            `yaml:"name"`
	Model         string            `yaml:"model"`
	DraftModel    string            `yaml:"draft_model,omitempty"`
	ContextSize   int               `yaml:"context_size,omitempty"`
	Threads       int               `yaml:"threads,omitempty"`
	ServerOptions map[string]string `yaml:"server_options,omitempty"`
}

// Preset represents a model + argument combination.
type Preset struct {
	Name             string            `yaml:"name"`                         // Required, used as identifier
	Model            string            `yaml:"model,omitempty"`              // single mode only
	DraftModel       string            `yaml:"draft_model,omitempty"`        // single mode only
	ContextSize      int               `yaml:"context_size,omitempty"`       // single mode only
	Threads          int               `yaml:"threads,omitempty"`            // single mode only
	Port             int               `yaml:"port,omitempty"`               // shared
	Host             string            `yaml:"host,omitempty"`               // shared
	ExtraArgs        []string          `yaml:"extra_args,omitempty"`         // single mode only
	Mode             string            `yaml:"mode,omitempty"`               // "single" (default) or "router"
	ModelsMax        int               `yaml:"models_max,omitempty"`         // router mode only
	SleepIdleSeconds int               `yaml:"sleep_idle_seconds,omitempty"` // router mode only
	Models           []ModelEntry      `yaml:"models,omitempty"`             // router mode only
	ServerOptions    map[string]string `yaml:"server_options,omitempty"`     // router mode only, [*] section
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

// IsRouter returns true if this preset uses router mode.
func (p *Preset) IsRouter() bool {
	return p.Mode == "router"
}

// BuildArgs builds the command-line arguments for llama-server in single mode.
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

// BuildRouterArgs builds the command-line arguments for llama-server in router mode.
func (p *Preset) BuildRouterArgs(configPath string) []string {
	args := []string{
		"--models-preset", configPath,
		"--port", strconv.Itoa(p.GetPort()),
		"--host", p.GetHost(),
	}

	if p.ModelsMax > 0 {
		args = append(args, "--models-max", strconv.Itoa(p.ModelsMax))
	}

	if p.SleepIdleSeconds > 0 {
		args = append(args, "--sleep-idle-seconds", strconv.Itoa(p.SleepIdleSeconds))
	}

	return args
}

// GenerateConfigINI generates config.ini content for router mode.
// The returned string is ready to be written to a file.
func (p *Preset) GenerateConfigINI() string {
	var b strings.Builder

	// [*] global section from top-level ServerOptions
	if len(p.ServerOptions) > 0 {
		b.WriteString("[*]\n")
		for _, k := range slices.Sorted(maps.Keys(p.ServerOptions)) {
			fmt.Fprintf(&b, "%s = %s\n", k, p.ServerOptions[k])
		}
		b.WriteString("\n")
	}

	// Per-model sections
	for _, m := range p.Models {
		fmt.Fprintf(&b, "[%s]\n", m.Name)

		modelPath := strings.TrimPrefix(m.Model, "f:")
		fmt.Fprintf(&b, "model = %s\n", modelPath)

		if m.DraftModel != "" {
			draftPath := strings.TrimPrefix(m.DraftModel, "f:")
			fmt.Fprintf(&b, "model-draft = %s\n", draftPath)
		}

		if m.ContextSize > 0 {
			fmt.Fprintf(&b, "ctx-size = %d\n", m.ContextSize)
		}

		if m.Threads > 0 {
			fmt.Fprintf(&b, "threads = %d\n", m.Threads)
		}

		if len(m.ServerOptions) > 0 {
			for _, k := range slices.Sorted(maps.Keys(m.ServerOptions)) {
				fmt.Fprintf(&b, "%s = %s\n", k, m.ServerOptions[k])
			}
		}

		b.WriteString("\n")
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}

// Validate checks that the preset configuration is consistent.
func (p *Preset) Validate() error {
	mode := p.Mode
	if mode == "" {
		mode = "single"
	}

	if mode != "single" && mode != "router" {
		return fmt.Errorf("mode must be 'single' or 'router'")
	}

	if mode == "router" {
		return p.validateRouter()
	}
	return p.validateSingle()
}

func (p *Preset) validateSingle() error {
	if len(p.Models) > 0 {
		return fmt.Errorf("single mode uses 'model' field, not 'models' list")
	}
	if len(p.ServerOptions) > 0 {
		return fmt.Errorf("single mode uses 'extra_args' instead of 'server_options'")
	}
	if p.ModelsMax > 0 {
		return fmt.Errorf("models_max is only valid in router mode")
	}
	if p.SleepIdleSeconds > 0 {
		return fmt.Errorf("sleep_idle_seconds is only valid in router mode")
	}
	if p.Model == "" {
		return fmt.Errorf("model field is required")
	}
	if strings.ContainsAny(p.Model, "\n\r") {
		return fmt.Errorf("model field must not contain newline characters")
	}
	if p.DraftModel != "" && strings.ContainsAny(p.DraftModel, "\n\r") {
		return fmt.Errorf("draft_model field must not contain newline characters")
	}
	return nil
}

func (p *Preset) validateRouter() error {
	if p.Model != "" {
		return fmt.Errorf("router mode defines models in the 'models' list, not as a top-level field")
	}
	if len(p.ExtraArgs) > 0 {
		return fmt.Errorf("router mode uses 'server_options' instead of 'extra_args'")
	}
	if p.DraftModel != "" {
		return fmt.Errorf("router mode defines draft_model per model in the 'models' list, not as a top-level field")
	}
	if p.ContextSize > 0 {
		return fmt.Errorf("router mode defines context_size per model in the 'models' list, not as a top-level field")
	}
	if p.Threads > 0 {
		return fmt.Errorf("router mode defines threads per model in the 'models' list, not as a top-level field")
	}
	if len(p.Models) == 0 {
		return fmt.Errorf("at least one model is required for router mode")
	}

	if err := validateServerOptions(p.ServerOptions); err != nil {
		return err
	}

	seen := make(map[string]bool)
	for _, m := range p.Models {
		if err := ValidateName(m.Name); err != nil {
			return fmt.Errorf("invalid model name: %w", err)
		}
		if seen[m.Name] {
			return fmt.Errorf("duplicate model name: '%s'", m.Name)
		}
		seen[m.Name] = true

		if m.Model == "" {
			return fmt.Errorf("model field is required for model '%s'", m.Name)
		}

		// Validate model entry fields and check for duplicates with server_options
		if err := validateModelEntry(m); err != nil {
			return err
		}
	}

	return nil
}

func validateModelEntry(m ModelEntry) error {
	if strings.ContainsAny(m.Model, "\n\r") {
		return fmt.Errorf("model field must not contain newline characters")
	}
	if m.DraftModel != "" && strings.ContainsAny(m.DraftModel, "\n\r") {
		return fmt.Errorf("draft_model field must not contain newline characters")
	}

	if m.ContextSize > 0 {
		if _, ok := m.ServerOptions["ctx-size"]; ok {
			return fmt.Errorf("'context_size' and server_options 'ctx-size' cannot both be set; use one or the other")
		}
	}
	if m.Threads > 0 {
		if _, ok := m.ServerOptions["threads"]; ok {
			return fmt.Errorf("'threads' and server_options 'threads' cannot both be set; use one or the other")
		}
	}
	if m.DraftModel != "" {
		if _, ok := m.ServerOptions["model-draft"]; ok {
			return fmt.Errorf("'draft_model' and server_options 'model-draft' cannot both be set; use one or the other")
		}
	}
	if _, ok := m.ServerOptions["model"]; ok {
		return fmt.Errorf("'model' field and server_options 'model' cannot both be set; use one or the other")
	}

	return validateServerOptions(m.ServerOptions)
}

// validateServerOptions checks that server_options keys and values do not contain newline characters.
func validateServerOptions(opts map[string]string) error {
	for k, v := range opts {
		if strings.ContainsAny(k, "\n\r") {
			return fmt.Errorf("server_options key must not contain newline characters")
		}
		if strings.ContainsAny(v, "\n\r") {
			return fmt.Errorf("server_options value must not contain newline characters")
		}
	}
	return nil
}
