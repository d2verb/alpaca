package preset

import (
	"slices"
	"testing"
)

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
