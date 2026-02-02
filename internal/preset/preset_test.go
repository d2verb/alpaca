package preset

import (
	"slices"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"already valid", "my-preset", "my-preset"},
		{"with spaces", "my preset name", "my-preset-name"},
		{"with special chars", "my@preset!name", "my-preset-name"},
		{"leading/trailing special", "!@#preset$%^", "preset"},
		{"consecutive special chars", "my...preset---name", "my-preset-name"},
		{"underscores preserved", "my_preset_name", "my_preset_name"},
		{"mixed valid chars", "My-Preset_123", "My-Preset_123"},
		{"dots replaced", "my.preset.name", "my-preset-name"},
		{"slashes replaced", "path/to/preset", "path-to-preset"},
		{"empty string", "", ""},
		{"all invalid chars", "!@#$%^&*()", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid alphanumeric", "mypreset", false},
		{"valid with numbers", "preset123", false},
		{"valid with hyphen", "my-preset", false},
		{"valid with underscore", "my_preset", false},
		{"valid mixed", "My-Preset_123", false},
		{"empty string", "", true},
		{"contains space", "my preset", true},
		{"contains dot", "my.preset", true},
		{"contains exclamation", "preset!", true},
		{"contains at sign", "preset@name", true},
		{"contains slash", "org/preset", true},
		{"contains colon", "preset:name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestPreset_GetPort(t *testing.T) {
	tests := []struct {
		name string
		port int
		want int
	}{
		{"returns custom port", 9090, 9090},
		{"returns default when zero", 0, DefaultPort},
		{"returns default when negative", -1, DefaultPort},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preset{Port: tt.port}
			if got := p.GetPort(); got != tt.want {
				t.Errorf("GetPort() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPreset_GetHost(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{"returns custom host", "0.0.0.0", "0.0.0.0"},
		{"returns default when empty", "", DefaultHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preset{Host: tt.host}
			if got := p.GetHost(); got != tt.want {
				t.Errorf("GetHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreset_GetContextSize(t *testing.T) {
	tests := []struct {
		name        string
		contextSize int
		want        int
	}{
		{"returns custom context size", 4096, 4096},
		{"returns default when zero", 0, DefaultContextSize},
		{"returns default when negative", -1, DefaultContextSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preset{ContextSize: tt.contextSize}
			if got := p.GetContextSize(); got != tt.want {
				t.Errorf("GetContextSize() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPreset_GetGPULayers(t *testing.T) {
	tests := []struct {
		name      string
		gpuLayers int
		want      int
	}{
		{"returns custom gpu layers", 32, 32},
		{"returns default when not set", 0, DefaultGPULayers},
		// Note: Cannot distinguish between "not set in YAML" and "explicitly set to 0".
		// To use CPU-only mode (0 GPU layers), use extra_args: ["--n-gpu-layers", "0"]
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preset{GPULayers: tt.gpuLayers}
			if got := p.GetGPULayers(); got != tt.want {
				t.Errorf("GetGPULayers() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPreset_Endpoint(t *testing.T) {
	tests := []struct {
		name string
		host string
		port int
		want string
	}{
		{"with defaults", "", 0, "http://127.0.0.1:8080"},
		{"with custom host", "0.0.0.0", 0, "http://0.0.0.0:8080"},
		{"with custom port", "", 9090, "http://127.0.0.1:9090"},
		{"with custom host and port", "localhost", 3000, "http://localhost:3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preset{Host: tt.host, Port: tt.port}
			if got := p.Endpoint(); got != tt.want {
				t.Errorf("Endpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPreset_BuildArgs(t *testing.T) {
	tests := []struct {
		name   string
		preset Preset
		want   []string
	}{
		{
			name:   "minimal preset",
			preset: Preset{Model: "/path/to/model.gguf"},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "-1",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with context size",
			preset: Preset{
				Model:       "/path/to/model.gguf",
				ContextSize: 4096,
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "4096",
				"--n-gpu-layers", "-1",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with gpu layers",
			preset: Preset{
				Model:     "/path/to/model.gguf",
				GPULayers: 32,
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "32",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with threads",
			preset: Preset{
				Model:   "/path/to/model.gguf",
				Threads: 8,
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "-1",
				"--threads", "8",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with custom port and host",
			preset: Preset{
				Model: "/path/to/model.gguf",
				Port:  9090,
				Host:  "0.0.0.0",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "-1",
				"--port", "9090",
				"--host", "0.0.0.0",
			},
		},
		{
			name: "with extra args",
			preset: Preset{
				Model:     "/path/to/model.gguf",
				ExtraArgs: []string{"--verbose", "--log-disable"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "-1",
				"--port", "8080",
				"--host", "127.0.0.1",
				"--verbose", "--log-disable",
			},
		},
		{
			name: "with space-separated extra args",
			preset: Preset{
				Model:     "/path/to/model.gguf",
				ExtraArgs: []string{"-b 2048", "-ub 2048", "--jinja"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "-1",
				"--port", "8080",
				"--host", "127.0.0.1",
				"-b", "2048", "-ub", "2048", "--jinja",
			},
		},
		{
			name: "with mixed extra args formats",
			preset: Preset{
				Model:     "/path/to/model.gguf",
				ExtraArgs: []string{"-b", "2048", "--temp 0.7", "--jinja"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "-1",
				"--port", "8080",
				"--host", "127.0.0.1",
				"-b", "2048", "--temp", "0.7", "--jinja",
			},
		},
		{
			name: "full preset",
			preset: Preset{
				Model:       "/path/to/model.gguf",
				ContextSize: 2048,
				GPULayers:   16,
				Threads:     4,
				Port:        3000,
				Host:        "localhost",
				ExtraArgs:   []string{"--mlock"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "2048",
				"--n-gpu-layers", "16",
				"--threads", "4",
				"--port", "3000",
				"--host", "localhost",
				"--mlock",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.preset.BuildArgs()
			if !slices.Equal(got, tt.want) {
				t.Errorf("BuildArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
