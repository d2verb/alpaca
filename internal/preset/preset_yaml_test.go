package preset

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

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
