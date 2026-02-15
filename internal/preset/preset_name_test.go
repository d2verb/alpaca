package preset

import (
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
