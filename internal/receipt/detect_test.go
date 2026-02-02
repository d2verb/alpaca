package receipt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstallSource(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		path     string
		envVars  map[string]string
		expected InstallSource
	}{
		{
			name:     "Homebrew Apple Silicon",
			path:     "/opt/homebrew/bin/alpaca",
			expected: SourceBrew,
		},
		{
			name:     "Homebrew Intel Cellar",
			path:     "/usr/local/Cellar/alpaca/1.0.0/bin/alpaca",
			expected: SourceBrew,
		},
		{
			name:     "Homebrew Intel opt",
			path:     "/usr/local/opt/alpaca/bin/alpaca",
			expected: SourceBrew,
		},
		{
			name:     "Linuxbrew",
			path:     "/home/linuxbrew/.linuxbrew/bin/alpaca",
			expected: SourceBrew,
		},
		{
			name:     "Custom HOMEBREW_PREFIX",
			path:     "/custom/homebrew/bin/alpaca",
			envVars:  map[string]string{"HOMEBREW_PREFIX": "/custom/homebrew"},
			expected: SourceBrew,
		},
		{
			name:     "HOMEBREW_PREFIX with trailing slash",
			path:     "/custom/homebrew/bin/alpaca",
			envVars:  map[string]string{"HOMEBREW_PREFIX": "/custom/homebrew/"},
			expected: SourceBrew,
		},
		{
			name:     "Not Homebrew with similar prefix",
			path:     "/opt/homebrew2/bin/alpaca",
			expected: SourceUnknown,
		},
		{
			name:     "Go install with GOBIN",
			path:     "/custom/gobin/alpaca",
			envVars:  map[string]string{"GOBIN": "/custom/gobin"},
			expected: SourceGo,
		},
		{
			name:     "Go install with GOPATH",
			path:     "/custom/gopath/bin/alpaca",
			envVars:  map[string]string{"GOPATH": "/custom/gopath"},
			expected: SourceGo,
		},
		{
			name:     "Not Go with similar prefix",
			path:     "/custom/gobin2/alpaca",
			envVars:  map[string]string{"GOBIN": "/custom/gobin"},
			expected: SourceUnknown,
		},
		{
			name:     "System bin unknown",
			path:     "/usr/bin/alpaca",
			expected: SourceUnknown,
		},
		{
			name:     "System local bin unknown",
			path:     "/usr/local/bin/alpaca",
			expected: SourceUnknown,
		},
		{
			name:     "User local bin script",
			path:     filepath.Join(home, ".local", "bin", "alpaca"),
			expected: SourceScript,
		},
		{
			name:     "User bin script",
			path:     filepath.Join(home, "bin", "alpaca"),
			expected: SourceScript,
		},
		{
			name:     "Random path unknown",
			path:     "/some/random/path/alpaca",
			expected: SourceUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and clear environment variables
			savedEnv := make(map[string]string)
			envsToClear := []string{"HOMEBREW_PREFIX", "GOBIN", "GOPATH"}
			for _, env := range envsToClear {
				savedEnv[env] = os.Getenv(env)
				os.Unsetenv(env)
			}
			defer func() {
				for k, v := range savedEnv {
					if v != "" {
						os.Setenv(k, v)
					}
				}
			}()

			// Set test-specific environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := DetectInstallSource(tt.path)
			if got != tt.expected {
				t.Errorf("DetectInstallSource(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsHomebrewPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "Apple Silicon prefix",
			path:     "/opt/homebrew/Cellar/alpaca/1.0/bin/alpaca",
			expected: true,
		},
		{
			name:     "Intel Cellar",
			path:     "/usr/local/Cellar/alpaca/1.0/bin/alpaca",
			expected: true,
		},
		{
			name:     "Not Homebrew",
			path:     "/usr/bin/alpaca",
			expected: false,
		},
		{
			name:     "HOMEBREW_PREFIX set",
			path:     "/my/brew/bin/alpaca",
			envVars:  map[string]string{"HOMEBREW_PREFIX": "/my/brew"},
			expected: true,
		},
		{
			name:     "Similar prefix but not Homebrew",
			path:     "/opt/homebrew2/bin/alpaca",
			expected: false,
		},
		{
			name:     "Exact prefix match",
			path:     "/opt/homebrew",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore HOMEBREW_PREFIX
			saved := os.Getenv("HOMEBREW_PREFIX")
			os.Unsetenv("HOMEBREW_PREFIX")
			defer func() {
				if saved != "" {
					os.Setenv("HOMEBREW_PREFIX", saved)
				}
			}()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := isHomebrewPath(tt.path)
			if got != tt.expected {
				t.Errorf("isHomebrewPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsGoInstallPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		path     string
		envVars  map[string]string
		expected bool
	}{
		{
			name:     "Default GOPATH",
			path:     filepath.Join(home, "go", "bin", "alpaca"),
			expected: true,
		},
		{
			name:     "Custom GOPATH",
			path:     "/custom/go/bin/alpaca",
			envVars:  map[string]string{"GOPATH": "/custom/go"},
			expected: true,
		},
		{
			name:     "GOBIN set",
			path:     "/my/gobin/alpaca",
			envVars:  map[string]string{"GOBIN": "/my/gobin"},
			expected: true,
		},
		{
			name:     "Not Go path",
			path:     "/usr/bin/alpaca",
			expected: false,
		},
		{
			name:     "Similar prefix but not Go",
			path:     "/my/gobin2/alpaca",
			envVars:  map[string]string{"GOBIN": "/my/gobin"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env vars
			savedGOBIN := os.Getenv("GOBIN")
			savedGOPATH := os.Getenv("GOPATH")
			os.Unsetenv("GOBIN")
			os.Unsetenv("GOPATH")
			defer func() {
				if savedGOBIN != "" {
					os.Setenv("GOBIN", savedGOBIN)
				}
				if savedGOPATH != "" {
					os.Setenv("GOPATH", savedGOPATH)
				}
			}()

			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			got := isGoInstallPath(tt.path)
			if got != tt.expected {
				t.Errorf("isGoInstallPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsUserLocalPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "User .local/bin",
			path:     filepath.Join(home, ".local", "bin", "alpaca"),
			expected: true,
		},
		{
			name:     "User bin",
			path:     filepath.Join(home, "bin", "alpaca"),
			expected: true,
		},
		{
			name:     "System bin",
			path:     "/usr/bin/alpaca",
			expected: false,
		},
		{
			name:     "Random path",
			path:     "/some/path/alpaca",
			expected: false,
		},
		{
			name:     "Similar prefix but not user local",
			path:     filepath.Join(home, ".local", "bin2", "alpaca"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUserLocalPath(tt.path)
			if got != tt.expected {
				t.Errorf("isUserLocalPath(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestHasPathPrefix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected bool
	}{
		{
			name:     "Exact match",
			path:     "/opt/homebrew",
			prefix:   "/opt/homebrew",
			expected: true,
		},
		{
			name:     "Proper prefix",
			path:     "/opt/homebrew/bin/alpaca",
			prefix:   "/opt/homebrew",
			expected: true,
		},
		{
			name:     "False positive avoided",
			path:     "/opt/homebrew2/bin/alpaca",
			prefix:   "/opt/homebrew",
			expected: false,
		},
		{
			name:     "No match",
			path:     "/usr/bin/alpaca",
			prefix:   "/opt/homebrew",
			expected: false,
		},
		{
			name:     "Nested path",
			path:     "/home/user/.local/bin/alpaca",
			prefix:   "/home/user/.local/bin",
			expected: true,
		},
		{
			name:     "Prefix with trailing slash",
			path:     "/opt/homebrew/bin/alpaca",
			prefix:   "/opt/homebrew/",
			expected: true,
		},
		{
			name:     "Empty prefix",
			path:     "/opt/homebrew/bin/alpaca",
			prefix:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPathPrefix(tt.path, tt.prefix)
			if got != tt.expected {
				t.Errorf("hasPathPrefix(%q, %q) = %v, want %v", tt.path, tt.prefix, got, tt.expected)
			}
		})
	}
}
