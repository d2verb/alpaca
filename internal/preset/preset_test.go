package preset

import (
	"testing"
)

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
				"--port", "8080",
				"--host", "127.0.0.1",
				"--verbose", "--log-disable",
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
			if !slicesEqual(got, tt.want) {
				t.Errorf("BuildArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
