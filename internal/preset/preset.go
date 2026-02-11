// Package preset handles preset management.
package preset

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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
)

// reservedOptionsKeys are keys that cannot be used in the top-level options map.
var reservedOptionsKeys = []string{
	"port", "host", "model", "model-draft", "mmproj", "models-max", "sleep-idle-seconds",
}

// reservedModelEntryOptionsKeys are keys that cannot be used in ModelEntry options.
var reservedModelEntryOptionsKeys = []string{
	"port", "host", "model", "model-draft", "mmproj",
}

// Options is a map of llama-server options.
// YAML scalars (string/int/float/bool) are accepted and stored as strings.
// Non-scalar values (lists, maps) and null values are rejected.
type Options map[string]string

// UnmarshalYAML uses yaml.Node to normalize all scalar values to strings.
// !!bool values are normalized to lowercase "true"/"false".
func (o *Options) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("options must be a mapping")
	}
	*o = make(Options, len(value.Content)/2)
	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]
		if keyNode.Kind != yaml.ScalarNode || valNode.Kind != yaml.ScalarNode {
			return fmt.Errorf("options key and value must be scalars")
		}
		if valNode.Tag == "!!null" {
			return fmt.Errorf("options key %q: value must not be null", keyNode.Value)
		}
		val := valNode.Value
		if valNode.Tag == "!!bool" {
			val = strings.ToLower(val)
		}
		(*o)[keyNode.Value] = val
	}
	return nil
}

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
	Name       string  `yaml:"name"`
	Model      string  `yaml:"model"`
	DraftModel string  `yaml:"draft-model,omitempty"`
	Mmproj     string  `yaml:"mmproj,omitempty" json:"mmproj,omitempty"`
	Options    Options `yaml:"options,omitempty"`
}

// Preset represents a model + argument combination.
type Preset struct {
	Name        string       `yaml:"name"`
	Model       string       `yaml:"model,omitempty"`
	DraftModel  string       `yaml:"draft-model,omitempty"`
	Mmproj      string       `yaml:"mmproj,omitempty" json:"mmproj,omitempty"`
	Mode        string       `yaml:"mode,omitempty"`
	Port        int          `yaml:"port,omitempty"`
	Host        string       `yaml:"host,omitempty"`
	MaxModels   int          `yaml:"max-models,omitempty"`
	IdleTimeout int          `yaml:"idle-timeout,omitempty"`
	Options     Options      `yaml:"options,omitempty"`
	Models      []ModelEntry `yaml:"models,omitempty"`
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

// Endpoint returns the HTTP endpoint for this preset.
func (p *Preset) Endpoint() string {
	return fmt.Sprintf("http://%s:%d", p.GetHost(), p.GetPort())
}

// IsRouter returns true if this preset uses router mode.
func (p *Preset) IsRouter() bool {
	return p.Mode == "router"
}

// IsMmprojActive reports whether the mmproj value represents an active mmproj
// configuration (i.e. not empty and not "none").
func IsMmprojActive(mmproj string) bool {
	return mmproj != "" && mmproj != "none"
}

// BuildArgs builds the command-line arguments for llama-server in single mode.
func (p *Preset) BuildArgs() []string {
	modelPath := strings.TrimPrefix(p.Model, "f:")

	args := []string{"-m", modelPath}

	if p.DraftModel != "" {
		draftModelPath := strings.TrimPrefix(p.DraftModel, "f:")
		args = append(args, "--model-draft", draftModelPath)
	}

	if IsMmprojActive(p.Mmproj) {
		mmprojPath := strings.TrimPrefix(p.Mmproj, "f:")
		args = append(args, "--mmproj", mmprojPath)
	}

	args = append(args, "--port", strconv.Itoa(p.GetPort()))
	args = append(args, "--host", p.GetHost())

	// Convert options map to CLI args (sorted by key)
	for _, k := range slices.Sorted(maps.Keys(p.Options)) {
		v := p.Options[k]
		switch v {
		case "true":
			args = append(args, "--"+k)
		case "false":
			// skip
		default:
			args = append(args, "--"+k, v)
		}
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

	if p.MaxModels > 0 {
		args = append(args, "--models-max", strconv.Itoa(p.MaxModels))
	}

	if p.IdleTimeout > 0 {
		args = append(args, "--sleep-idle-seconds", strconv.Itoa(p.IdleTimeout))
	}

	return args
}

// GenerateConfigINI generates config.ini content for router mode.
// The returned string is ready to be written to a file.
func (p *Preset) GenerateConfigINI() string {
	var b strings.Builder

	// [*] global section from top-level Options
	if len(p.Options) > 0 {
		b.WriteString("[*]\n")
		for _, k := range slices.Sorted(maps.Keys(p.Options)) {
			fmt.Fprintf(&b, "%s = %s\n", k, p.Options[k])
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

		if IsMmprojActive(m.Mmproj) {
			mmprojPath := strings.TrimPrefix(m.Mmproj, "f:")
			fmt.Fprintf(&b, "mmproj = %s\n", mmprojPath)
		}

		if len(m.Options) > 0 {
			for _, k := range slices.Sorted(maps.Keys(m.Options)) {
				fmt.Fprintf(&b, "%s = %s\n", k, m.Options[k])
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
	if p.MaxModels > 0 {
		return fmt.Errorf("max-models is only valid in router mode")
	}
	if p.IdleTimeout > 0 {
		return fmt.Errorf("idle-timeout is only valid in router mode")
	}
	if p.Model == "" {
		return fmt.Errorf("model field is required")
	}
	if strings.ContainsAny(p.Model, "\n\r") {
		return fmt.Errorf("model field must not contain newline characters")
	}
	if p.DraftModel != "" && strings.ContainsAny(p.DraftModel, "\n\r") {
		return fmt.Errorf("draft-model field must not contain newline characters")
	}
	if err := validateMmproj(p.Mmproj); err != nil {
		return err
	}
	return validateOptions(p.Options, reservedOptionsKeys)
}

func (p *Preset) validateRouter() error {
	if p.Model != "" {
		return fmt.Errorf("router mode defines models in the 'models' list, not as a top-level field")
	}
	if p.DraftModel != "" {
		return fmt.Errorf("router mode defines draft-model per model in the 'models' list, not as a top-level field")
	}
	if p.Mmproj != "" {
		return fmt.Errorf("router mode defines mmproj per model in the 'models' list, not as a top-level field")
	}
	if len(p.Models) == 0 {
		return fmt.Errorf("at least one model is required for router mode")
	}

	if err := validateOptions(p.Options, reservedOptionsKeys); err != nil {
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
		return fmt.Errorf("draft-model field must not contain newline characters")
	}
	if err := validateMmproj(m.Mmproj); err != nil {
		return err
	}

	return validateOptions(m.Options, reservedModelEntryOptionsKeys)
}

// validateMmproj validates the mmproj field value.
// Valid values: empty (omitted), "none" (lowercase only), or "f:" prefixed path.
func validateMmproj(mmproj string) error {
	if mmproj == "" {
		return nil
	}
	if mmproj == "none" {
		return nil
	}
	if strings.ContainsAny(mmproj, "\n\r") {
		return fmt.Errorf("mmproj field must not contain newline characters")
	}
	if strings.HasPrefix(mmproj, "f:") {
		return nil
	}
	return fmt.Errorf("invalid mmproj value: got %q; expected 'none', 'f:/path', or omit", mmproj)
}

// validateOptions checks that options keys are not reserved and do not contain newline characters.
func validateOptions(opts Options, reserved []string) error {
	for k, v := range opts {
		if strings.ContainsAny(k, "\n\r") {
			return fmt.Errorf("options key must not contain newline characters")
		}
		if strings.ContainsAny(v, "\n\r") {
			return fmt.Errorf("options value must not contain newline characters")
		}
		if slices.Contains(reserved, k) {
			return fmt.Errorf("options key %q is reserved and cannot be used in options; use the top-level %q field instead", k, k)
		}
	}
	return nil
}
