package main

import "testing"

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 500, "500 B"},
		{"one KB", 1024, "1.0 KB"},
		{"kilobytes", 1536, "1.5 KB"},
		{"one MB", 1024 * 1024, "1.0 MB"},
		{"megabytes", 1536 * 1024, "1.5 MB"},
		{"one GB", 1024 * 1024 * 1024, "1.0 GB"},
		{"gigabytes", 4831838208, "4.5 GB"},  // 4.5 * 1024^3
		{"large file", 8375319756, "7.8 GB"}, // 7.8 * 1024^3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}
