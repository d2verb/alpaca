package preset

import (
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
			name: "with draft model",
			preset: Preset{
				Model:      "/path/to/model.gguf",
				DraftModel: "f:/path/to/draft.gguf",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--model-draft", "/path/to/draft.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with boolean true option becomes flag",
			preset: Preset{
				Model:   "/path/to/model.gguf",
				Options: Options{"mlock": "true"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
				"--mlock",
			},
		},
		{
			name: "with boolean false option is skipped",
			preset: Preset{
				Model:   "/path/to/model.gguf",
				Options: Options{"mlock": "false"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with value options",
			preset: Preset{
				Model:   "/path/to/model.gguf",
				Options: Options{"ctx-size": "4096", "threads": "8"},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
				"--ctx-size", "4096",
				"--threads", "8",
			},
		},
		{
			name: "options are sorted alphabetically",
			preset: Preset{
				Model: "/path/to/model.gguf",
				Options: Options{
					"threads":    "8",
					"ctx-size":   "4096",
					"flash-attn": "on",
					"mlock":      "true",
					"no-mmap":    "true",
					"temp":       "0.7",
				},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
				"--ctx-size", "4096",
				"--flash-attn", "on",
				"--mlock",
				"--no-mmap",
				"--temp", "0.7",
				"--threads", "8",
			},
		},
		{
			name: "full preset with all features",
			preset: Preset{
				Model:      "/path/to/model.gguf",
				DraftModel: "f:/path/to/draft.gguf",
				Port:       3000,
				Host:       "localhost",
				Options: Options{
					"ctx-size": "2048",
					"threads":  "4",
					"mlock":    "true",
				},
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--model-draft", "/path/to/draft.gguf",
				"--port", "3000",
				"--host", "localhost",
				"--ctx-size", "2048",
				"--mlock",
				"--threads", "4",
			},
		},
		{
			name: "with mmproj",
			preset: Preset{
				Model:  "/path/to/model.gguf",
				Mmproj: "f:/path/to/mmproj.gguf",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--mmproj", "/path/to/mmproj.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with mmproj empty does not add flag",
			preset: Preset{
				Model:  "/path/to/model.gguf",
				Mmproj: "",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
			},
		},
		{
			name: "with mmproj none does not add flag",
			preset: Preset{
				Model:  "/path/to/model.gguf",
				Mmproj: "none",
			},
			want: []string{
				"-m", "/path/to/model.gguf",
				"--port", "8080",
				"--host", "127.0.0.1",
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
			name: "with max-models",
			preset: Preset{
				Mode:      "router",
				MaxModels: 3,
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
			name: "with idle-timeout",
			preset: Preset{
				Mode:        "router",
				IdleTimeout: 300,
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
				Mode:        "router",
				Port:        9090,
				Host:        "0.0.0.0",
				MaxModels:   2,
				IdleTimeout: 60,
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
			name: "single model with global options",
			preset: Preset{
				Mode:    "router",
				Options: Options{"ctx-size": "4096"},
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
						Name:    "llama",
						Model:   "f:/path/to/llama.gguf",
						Options: Options{"ctx-size": "2048", "threads": "4"},
					},
					{
						Name:    "codellama",
						Model:   "f:/path/to/codellama.gguf",
						Options: Options{"gpu-layers": "32"},
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
				Options: Options{
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
		{
			name: "model with mmproj",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:   "vision",
						Model:  "f:/path/to/model.gguf",
						Mmproj: "f:/path/to/mmproj.gguf",
					},
				},
			},
			want: "[vision]\nmodel = /path/to/model.gguf\nmmproj = /path/to/mmproj.gguf\n",
		},
		{
			name: "model without mmproj omits line",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "text-only", Model: "f:/path/to/model.gguf"},
				},
			},
			want: "[text-only]\nmodel = /path/to/model.gguf\n",
		},
		{
			name: "model with mmproj none omits line",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "no-vision", Model: "f:/path/to/model.gguf", Mmproj: "none"},
				},
			},
			want: "[no-vision]\nmodel = /path/to/model.gguf\n",
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
			name: "valid single mode with options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"ctx-size": "4096", "mlock": "true"},
			},
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
			name: "single mode with max-models",
			preset: Preset{
				Model:     "f:/path/to/model.gguf",
				MaxModels: 3,
			},
			wantErr: "max-models is only valid in router mode",
		},
		{
			name: "single mode with idle-timeout",
			preset: Preset{
				Model:       "f:/path/to/model.gguf",
				IdleTimeout: 300,
			},
			wantErr: "idle-timeout is only valid in router mode",
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
			name: "router mode with top-level draft-model",
			preset: Preset{
				Mode:       "router",
				DraftModel: "f:/draft.gguf",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode defines draft-model per model in the 'models' list, not as a top-level field",
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
			name: "router mode with newline in global options value",
			preset: Preset{
				Mode:    "router",
				Options: Options{"key": "val\nue"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "options value must not contain newline characters",
		},
		{
			name: "router mode with newline in global options key",
			preset: Preset{
				Mode:    "router",
				Options: Options{"ke\ny": "value"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "options key must not contain newline characters",
		},
		{
			name: "router mode with carriage return in global options key",
			preset: Preset{
				Mode:    "router",
				Options: Options{"ke\ry": "value"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "options key must not contain newline characters",
		},
		{
			name: "router mode with newline in model options value",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"key": "val\nue"},
					},
				},
			},
			wantErr: "options value must not contain newline characters",
		},
		{
			name: "router mode with newline in model options key",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"ke\ny": "value"},
					},
				},
			},
			wantErr: "options key must not contain newline characters",
		},
		{
			name: "router mode with carriage return in model options key",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"ke\ry": "value"},
					},
				},
			},
			wantErr: "options key must not contain newline characters",
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
			name:    "single mode with newline in draft-model",
			preset:  Preset{Model: "f:/path/to/model.gguf", DraftModel: "f:/path\n/draft.gguf"},
			wantErr: "draft-model field must not contain newline characters",
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
			name: "router mode with newline in draft-model path",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf", DraftModel: "f:/draft\n.gguf"},
				},
			},
			wantErr: "draft-model field must not contain newline characters",
		},
		// mmproj validation tests
		{
			name:   "single mode with empty mmproj is valid",
			preset: Preset{Model: "f:/path/to/model.gguf", Mmproj: ""},
		},
		{
			name:   "single mode with mmproj none is valid",
			preset: Preset{Model: "f:/path/to/model.gguf", Mmproj: "none"},
		},
		{
			name:   "single mode with mmproj f: path is valid",
			preset: Preset{Model: "f:/path/to/model.gguf", Mmproj: "f:/path/to/mmproj.gguf"},
		},
		{
			name:    "single mode with mmproj None is invalid",
			preset:  Preset{Model: "f:/path/to/model.gguf", Mmproj: "None"},
			wantErr: `invalid mmproj value: got "None"`,
		},
		{
			name:    "single mode with mmproj NONE is invalid",
			preset:  Preset{Model: "f:/path/to/model.gguf", Mmproj: "NONE"},
			wantErr: `invalid mmproj value: got "NONE"`,
		},
		{
			name:    "single mode with mmproj h: prefix is invalid",
			preset:  Preset{Model: "f:/path/to/model.gguf", Mmproj: "h:org/repo"},
			wantErr: `invalid mmproj value: got "h:org/repo"`,
		},
		{
			name:    "single mode with mmproj random string is invalid",
			preset:  Preset{Model: "f:/path/to/model.gguf", Mmproj: "random-string"},
			wantErr: `invalid mmproj value: got "random-string"`,
		},
		{
			name:    "single mode with mmproj containing newline is invalid",
			preset:  Preset{Model: "f:/path/to/model.gguf", Mmproj: "f:/path\n/mmproj.gguf"},
			wantErr: "mmproj field must not contain newline characters",
		},
		{
			name: "router mode with top-level mmproj is invalid",
			preset: Preset{
				Mode:   "router",
				Mmproj: "none",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: "router mode defines mmproj per model in the 'models' list, not as a top-level field",
		},
		{
			name: "router mode with per-model mmproj is valid",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf", Mmproj: "f:/path/to/mmproj.gguf"},
				},
			},
		},
		{
			name: "router mode with per-model mmproj none is valid",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf", Mmproj: "none"},
				},
			},
		},
		{
			name: "router mode with per-model invalid mmproj",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf", Mmproj: "random-string"},
				},
			},
			wantErr: `invalid mmproj value: got "random-string"`,
		},
		// Reserved key tests for top-level options
		{
			name: "single mode with reserved key mmproj in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"mmproj": "/path/to/mmproj.gguf"},
			},
			wantErr: `options key "mmproj" is reserved`,
		},
		{
			name: "router mode with reserved key mmproj in model options",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"mmproj": "/path/to/mmproj.gguf"},
					},
				},
			},
			wantErr: `options key "mmproj" is reserved`,
		},
		{
			name: "single mode with reserved key port in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"port": "8080"},
			},
			wantErr: `options key "port" is reserved`,
		},
		{
			name: "single mode with reserved key host in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"host": "127.0.0.1"},
			},
			wantErr: `options key "host" is reserved`,
		},
		{
			name: "single mode with reserved key model in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"model": "/other.gguf"},
			},
			wantErr: `options key "model" is reserved`,
		},
		{
			name: "single mode with reserved key model-draft in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"model-draft": "/draft.gguf"},
			},
			wantErr: `options key "model-draft" is reserved`,
		},
		{
			name: "single mode with reserved key models-max in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"models-max": "3"},
			},
			wantErr: `options key "models-max" is reserved`,
		},
		{
			name: "single mode with reserved key sleep-idle-seconds in options",
			preset: Preset{
				Model:   "f:/path/to/model.gguf",
				Options: Options{"sleep-idle-seconds": "300"},
			},
			wantErr: `options key "sleep-idle-seconds" is reserved`,
		},
		{
			name: "router mode with reserved key in global options",
			preset: Preset{
				Mode:    "router",
				Options: Options{"port": "8080"},
				Models: []ModelEntry{
					{Name: "llama", Model: "f:/llama.gguf"},
				},
			},
			wantErr: `options key "port" is reserved`,
		},
		// Reserved key tests for model entry options
		{
			name: "router mode with reserved key port in model options",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"port": "8080"},
					},
				},
			},
			wantErr: `options key "port" is reserved`,
		},
		{
			name: "router mode with reserved key model in model options",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"model": "/other.gguf"},
					},
				},
			},
			wantErr: `options key "model" is reserved`,
		},
		{
			name: "router mode with reserved key model-draft in model options",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"model-draft": "/draft.gguf"},
					},
				},
			},
			wantErr: `options key "model-draft" is reserved`,
		},
		// model entry options allow models-max and sleep-idle-seconds (not reserved at model level)
		{
			name: "router mode model options allow non-reserved keys",
			preset: Preset{
				Mode: "router",
				Models: []ModelEntry{
					{
						Name:    "llama",
						Model:   "f:/llama.gguf",
						Options: Options{"ctx-size": "4096", "threads": "8"},
					},
				},
			},
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

func TestOptions_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Options
		wantErr string
	}{
		{
			name:  "string values",
			input: "flash-attn: on\ncache-type-k: q8_0",
			want:  Options{"flash-attn": "on", "cache-type-k": "q8_0"},
		},
		{
			name:  "integer value stored as string",
			input: "ctx-size: 4096",
			want:  Options{"ctx-size": "4096"},
		},
		{
			name:  "float value stored as string",
			input: "temp: 0.7",
			want:  Options{"temp": "0.7"},
		},
		{
			name:  "bool true normalized to lowercase",
			input: "mlock: true",
			want:  Options{"mlock": "true"},
		},
		{
			name:  "bool True normalized to lowercase",
			input: "mlock: True",
			want:  Options{"mlock": "true"},
		},
		{
			name:  "bool TRUE normalized to lowercase",
			input: "mlock: TRUE",
			want:  Options{"mlock": "true"},
		},
		{
			name:  "bool false normalized to lowercase",
			input: "mlock: false",
			want:  Options{"mlock": "false"},
		},
		{
			name:  "bool FALSE normalized to lowercase",
			input: "mlock: FALSE",
			want:  Options{"mlock": "false"},
		},
		{
			name:  "on is treated as string not bool",
			input: "flash-attn: on",
			want:  Options{"flash-attn": "on"},
		},
		{
			name:  "off is treated as string not bool",
			input: "flash-attn: off",
			want:  Options{"flash-attn": "off"},
		},
		{
			name:  "yes is treated as string not bool",
			input: "flag: yes",
			want:  Options{"flag": "yes"},
		},
		{
			name:  "no is treated as string not bool",
			input: "flag: no",
			want:  Options{"flag": "no"},
		},
		{
			name:    "null value rejected",
			input:   "key: null",
			wantErr: `value must not be null`,
		},
		{
			name:    "null value rejected with tilde",
			input:   "key: ~",
			wantErr: `value must not be null`,
		},
		{
			name:    "list value rejected",
			input:   "key:\n  - a\n  - b",
			wantErr: "options key and value must be scalars",
		},
		{
			name:    "map value rejected",
			input:   "key:\n  nested: value",
			wantErr: "options key and value must be scalars",
		},
		{
			name:  "mixed types",
			input: "ctx-size: 4096\ntemp: 0.7\nmlock: true\nflash-attn: on\ncache-type-k: q8_0",
			want: Options{
				"ctx-size":     "4096",
				"temp":         "0.7",
				"mlock":        "true",
				"flash-attn":   "on",
				"cache-type-k": "q8_0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts Options
			err := yaml.Unmarshal([]byte(tt.input), &opts)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("UnmarshalYAML() expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("UnmarshalYAML() error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalYAML() unexpected error: %v", err)
			}

			if len(opts) != len(tt.want) {
				t.Fatalf("UnmarshalYAML() got %d keys, want %d", len(opts), len(tt.want))
			}

			for k, wantV := range tt.want {
				gotV, ok := opts[k]
				if !ok {
					t.Errorf("UnmarshalYAML() missing key %q", k)
					continue
				}
				if gotV != wantV {
					t.Errorf("UnmarshalYAML() key %q = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}
