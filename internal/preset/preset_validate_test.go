package preset

import (
	"strings"
	"testing"
)

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
			name:    "single mode with mmproj f: only (no path) is invalid",
			preset:  Preset{Model: "f:/path/to/model.gguf", Mmproj: "f:"},
			wantErr: "mmproj 'f:' prefix requires a path",
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
