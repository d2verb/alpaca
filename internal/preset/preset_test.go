package preset

import (
	"slices"
	"strings"
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
				"--ctx-size", "4096",
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
			name: "with threads",
			preset: Preset{
				Model:   "/path/to/model.gguf",
				Threads: 8,
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "4096",
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
				"--ctx-size", "4096",
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
				"--ctx-size", "4096",
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
				"--ctx-size", "4096",
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
				"--ctx-size", "4096",
				"--port", "8080",
				"--host", "127.0.0.1",
				"-b", "2048", "--temp", "0.7", "--jinja",
			},
		},
		{
			name: "with draft model",
			preset: Preset{
				Model:      "/path/to/model.gguf",
				DraftModel: "f:/path/to/draft.gguf",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--model-draft", "/path/to/draft.gguf",
				"--ctx-size", "4096",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with draft model without prefix",
			preset: Preset{
				Model:      "/path/to/model.gguf",
				DraftModel: "/path/to/draft.gguf",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--model-draft", "/path/to/draft.gguf",
				"--ctx-size", "4096",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "without draft model",
			preset: Preset{
				Model: "/path/to/model.gguf",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--ctx-size", "4096",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "full preset",
			preset: Preset{
				Model:       "/path/to/model.gguf",
				DraftModel:  "f:/path/to/draft.gguf",
				ContextSize: 2048,
				Threads:     4,
				Port:        3000,
				Host:        "localhost",
				ExtraArgs:   []string{"--mlock"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--model-draft", "/path/to/draft.gguf",
				"--ctx-size", "2048",
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

func TestPreset_IsRouter(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{"router mode", "router", true},
		{"single mode", "single", false},
		{"empty defaults to single", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Preset{Mode: tt.mode}
			if got := p.IsRouter(); got != tt.want {
				t.Errorf("IsRouter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreset_BuildRouterArgs(t *testing.T) {
	tests := []struct {
		name       string
		preset     Preset
		configPath string
		want       []string
	}{
		{
			name:       "minimal router preset",
			preset:     Preset{Mode: "router"},
			configPath: "/tmp/config.ini",
			want: []string{
				"--models-preset", "/tmp/config.ini",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with models_max",
			preset: Preset{
				Mode:      "router",
				ModelsMax: 3,
			},
			configPath: "/tmp/config.ini",
			want: []string{
				"--models-preset", "/tmp/config.ini",
				"--port", "8080",
				"--host", "127.0.0.1",
				"--models-max", "3",
			},
		},
		{
			name: "with sleep_idle_seconds",
			preset: Preset{
				Mode:             "router",
				SleepIdleSeconds: 300,
			},
			configPath: "/tmp/config.ini",
			want: []string{
				"--models-preset", "/tmp/config.ini",
				"--port", "8080",
				"--host", "127.0.0.1",
				"--sleep-idle-seconds", "300",
			},
		},
		{
			name: "with custom port and host",
			preset: Preset{
				Mode:             "router",
				Port:             9090,
				Host:             "0.0.0.0",
				ModelsMax:        2,
				SleepIdleSeconds: 60,
			},
			configPath: "/tmp/models.ini",
			want: []string{
				"--models-preset", "/tmp/models.ini",
				"--port", "9090",
				"--host", "0.0.0.0",
				"--models-max", "2",
				"--sleep-idle-seconds", "60",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.preset.BuildRouterArgs(tt.configPath)
			if !slices.Equal(got, tt.want) {
				t.Errorf("BuildRouterArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPreset_GenerateConfigINI(t *testing.T) {
	tests := []struct {
		name   string
		preset Preset
		want   string
	}{
		{
			name: "single model without global options",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/path/to/llama.gguf"},
				},
			},
			want: "[llama]\nmodel = /path/to/llama.gguf\n",
		},
		{
			name: "single model with global server_options",
			preset: Preset{
				Mode:          "router",
				ServerOptions: map[string]string{"ctx-size": "4096"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/path/to/llama.gguf"},
				},
			},
			want: "[*]\nctx-size = 4096\n\n[llama]\nmodel = /path/to/llama.gguf\n",
		},
		{
			name: "multiple models with per-model options",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:        "llama",
						Model:       "f:/path/to/llama.gguf",
						ContextSize: 2048,
						Threads:     4,
					},
					{
						Name:  "codellama",
						Model: "f:/path/to/codellama.gguf",
						ServerOptions: map[string]string{
							"gpu-layers": "32",
						},
					},
				},
			},
			want: "[llama]\nmodel = /path/to/llama.gguf\nctx-size = 2048\nthreads = 4\n\n[codellama]\nmodel = /path/to/codellama.gguf\ngpu-layers = 32\n",
		},
		{
			name: "model with draft model",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:       "llama",
						Model:      "f:/path/to/llama.gguf",
						DraftModel: "f:/path/to/draft.gguf",
					},
				},
			},
			want: "[llama]\nmodel = /path/to/llama.gguf\nmodel-draft = /path/to/draft.gguf\n",
		},
		{
			name: "global options sorted alphabetically",
			preset: Preset{
				Mode: "router",
				ServerOptions: map[string]string{
					"threads":  "8",
					"ctx-size": "4096",
					"mlock":    "true",
				},
				Models: []ModelEntry{
					{Name: "m1", Model: "/path/m1.gguf"},
				},
			},
			want: "[*]\nctx-size = 4096\nmlock = true\nthreads = 8\n\n[m1]\nmodel = /path/m1.gguf\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.preset.GenerateConfigINI()
			if got != tt.want {
				t.Errorf("GenerateConfigINI() =\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestPreset_Validate(t *testing.T) {
	tests := []struct {
		name    string
		preset  Preset
		wantErr string
	}{
		{
			name:   "valid single mode preset",
			preset: Preset{Model: "f:/path/to/model.gguf"},
		},
		{
			name:   "valid single mode with explicit mode",
			preset: Preset{Mode: "single", Model: "f:/path/to/model.gguf"},
		},
		{
			name: "valid router mode preset",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/path/to/llama.gguf"},
				},
			},
		},
		{
			name:    "invalid mode value",
			preset:  Preset{Mode: "cluster"},
			wantErr: "mode must be 'single' or 'router'",
		},
		{
			name:    "single mode missing model",
			preset:  Preset{},
			wantErr: "model field is required",
		},
		{
			name: "single mode with models list",
			preset: Preset{
				Model: "f:/path/to/model.gguf",
				Models: []ModelEntry{
					{Name: "extra", Model: "f:/extra.gguf"},
				},
			},
			wantErr: "single mode uses 'model' field, not 'models' list",
		},
		{
			name: "single mode with server_options",
			preset: Preset{
				Model:         "f:/path/to/model.gguf",
				ServerOptions: map[string]string{"key": "val"},
			},
			wantErr: "single mode uses 'extra_args' instead of 'server_options'",
		},
		{
			name: "router mode with top-level model",
			preset: Preset{
				Mode:  "router",
				Model: "f:/path/to/model.gguf",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode defines models in the 'models' list, not as a top-level field",
		},
		{
			name: "router mode with extra_args",
			preset: Preset{
				Mode:      "router",
				ExtraArgs: []string{"--verbose"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode uses 'server_options' instead of 'extra_args'",
		},
		{
			name: "router mode with top-level draft_model",
			preset: Preset{
				Mode:       "router",
				DraftModel: "f:/draft.gguf",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode defines draft_model per model in the 'models' list, not as a top-level field",
		},
		{
			name:    "router mode with no models",
			preset:  Preset{Mode: "router"},
			wantErr: "at least one model is required for router mode",
		},
		{
			name: "router mode with duplicate model names",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama1.gguf"},
					{Name: "llama", Model: "f:/llama2.gguf"},
				},
			},
			wantErr: "duplicate model name: 'llama'",
		},
		{
			name: "router mode with invalid model name",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "invalid name!", Model: "f:/model.gguf"},
				},
			},
			wantErr: "invalid model name",
		},
		{
			name: "router mode with missing model path",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama"},
				},
			},
			wantErr: "model field is required for model 'llama'",
		},
		{
			name: "router mode with newline in global server_options value",
			preset: Preset{
				Mode:          "router",
				ServerOptions: map[string]string{"key": "val\nue"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "server_options value must not contain newline characters",
		},
		{
			name: "router mode with newline in global server_options key",
			preset: Preset{
				Mode:          "router",
				ServerOptions: map[string]string{"ke\ny": "value"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "server_options key must not contain newline characters",
		},
		{
			name: "router mode with carriage return in global server_options key",
			preset: Preset{
				Mode:          "router",
				ServerOptions: map[string]string{"ke\ry": "value"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "server_options key must not contain newline characters",
		},
		{
			name: "router mode with newline in model server_options value",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						ServerOptions: map[string]string{"key": "val\nue"},
					},
				},
			},
			wantErr: "server_options value must not contain newline characters",
		},
		{
			name: "router mode with newline in model server_options key",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						ServerOptions: map[string]string{"ke\ny": "value"},
					},
				},
			},
			wantErr: "server_options key must not contain newline characters",
		},
		{
			name: "router mode with carriage return in model server_options key",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						ServerOptions: map[string]string{"ke\ry": "value"},
					},
				},
			},
			wantErr: "server_options key must not contain newline characters",
		},
		{
			name:    "single mode with newline in model",
			preset:  Preset{Model: "f:/path/to\n/model.gguf"},
			wantErr: "model field must not contain newline characters",
		},
		{
			name:    "single mode with carriage return in model",
			preset:  Preset{Model: "f:/path/to\r/model.gguf"},
			wantErr: "model field must not contain newline characters",
		},
		{
			name:    "single mode with newline in draft_model",
			preset:  Preset{Model: "f:/path/to/model.gguf", DraftModel: "f:/path\n/draft.gguf"},
			wantErr: "draft_model field must not contain newline characters",
		},
		{
			name: "router mode with newline in model path",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/path\n/llama.gguf"},
				},
			},
			wantErr: "model field must not contain newline characters",
		},
		{
			name: "router mode with newline in draft_model path",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf", DraftModel: "f:/draft\n.gguf"},
				},
			},
			wantErr: "draft_model field must not contain newline characters",
		},
		{
			name: "single mode with models_max",
			preset: Preset{
				Model:     "f:/path/to/model.gguf",
				ModelsMax: 3,
			},
			wantErr: "models_max is only valid in router mode",
		},
		{
			name: "single mode with sleep_idle_seconds",
			preset: Preset{
				Model:            "f:/path/to/model.gguf",
				SleepIdleSeconds: 300,
			},
			wantErr: "sleep_idle_seconds is only valid in router mode",
		},
		{
			name: "router mode with top-level context_size",
			preset: Preset{
				Mode:        "router",
				ContextSize: 4096,
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode defines context_size per model in the 'models' list, not as a top-level field",
		},
		{
			name: "router mode with top-level threads",
			preset: Preset{
				Mode:    "router",
				Threads: 8,
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode defines threads per model in the 'models' list, not as a top-level field",
		},
		{
			name: "model field and server_options model conflict",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						ServerOptions: map[string]string{"model": "/other.gguf"},
					},
				},
			},
			wantErr: "'model' field and server_options 'model' cannot both be set",
		},
		{
			name: "context_size and ctx-size server_option conflict",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						ContextSize:   4096,
						ServerOptions: map[string]string{"ctx-size": "2048"},
					},
				},
			},
			wantErr: "'context_size' and server_options 'ctx-size' cannot both be set",
		},
		{
			name: "threads and threads server_option conflict",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						Threads:       8,
						ServerOptions: map[string]string{"threads": "4"},
					},
				},
			},
			wantErr: "'threads' and server_options 'threads' cannot both be set",
		},
		{
			name: "draft_model and model-draft server_option conflict",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:          "llama",
						Model:         "f:/llama.gguf",
						DraftModel:    "f:/draft.gguf",
						ServerOptions: map[string]string{"model-draft": "/other.gguf"},
					},
				},
			},
			wantErr: "'draft_model' and server_options 'model-draft' cannot both be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.preset.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Validate() expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
