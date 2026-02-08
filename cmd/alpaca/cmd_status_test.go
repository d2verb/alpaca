package main

import "testing"

func TestStringVal(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		key  string
		want string
	}{
		{
			name: "existing string key",
			m:    map[string]any{"id": "qwen3"},
			key:  "id",
			want: "qwen3",
		},
		{
			name: "missing key returns empty",
			m:    map[string]any{"id": "qwen3"},
			key:  "status",
			want: "",
		},
		{
			name: "non-string value returns empty",
			m:    map[string]any{"count": 42},
			key:  "count",
			want: "",
		},
		{
			name: "empty map",
			m:    map[string]any{},
			key:  "id",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringVal(tt.m, tt.key)
			if got != tt.want {
				t.Errorf("stringVal(%v, %q) = %q, want %q", tt.m, tt.key, got, tt.want)
			}
		})
	}
}
