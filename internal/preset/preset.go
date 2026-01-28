// Package preset handles preset management.
package preset

import (
	"fmt"
	"strconv"
)

const (
	// DefaultPort is the default port for llama-server.
	DefaultPort = 8080
	// DefaultHost is the default host for llama-server.
	DefaultHost = "127.0.0.1"
)

// Preset represents a model + argument combination.
type Preset struct {
	Name        string   `yaml:"-"` // Derived from filename
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

// Endpoint returns the HTTP endpoint for this preset.
func (p *Preset) Endpoint() string {
	return fmt.Sprintf("http://%s:%d", p.GetHost(), p.GetPort())
}

// BuildArgs builds the command-line arguments for llama-server.
func (p *Preset) BuildArgs() []string {
	args := []string{
		"-m", p.Model,
	}

	if p.ContextSize > 0 {
		args = append(args, "--ctx-size", strconv.Itoa(p.ContextSize))
	}
	if p.GPULayers != 0 {
		args = append(args, "--n-gpu-layers", strconv.Itoa(p.GPULayers))
	}
	if p.Threads > 0 {
		args = append(args, "--threads", strconv.Itoa(p.Threads))
	}

	// Always include port and host with defaults
	args = append(args, "--port", strconv.Itoa(p.GetPort()))
	args = append(args, "--host", p.GetHost())

	args = append(args, p.ExtraArgs...)
	return args
}
