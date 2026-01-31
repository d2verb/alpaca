# Shell Completion Design

## Overview

Shell completion for Alpaca CLI using [kongplete](https://github.com/willabides/kongplete) (v0.4.0+).

### Target Completions

Commands that benefit from dynamic completion:

```bash
# Show command (presets and models only)
alpaca show <TAB>          # List all available presets and models
alpaca show p:<TAB>        # Complete preset names
alpaca show h:<TAB>        # Complete downloaded model identifiers

# Remove command (presets and models only)
alpaca rm <TAB>            # List all available presets and models
alpaca rm p:<TAB>          # Complete preset names
alpaca rm h:<TAB>          # Complete downloaded model identifiers

# Load command (presets, models, and files)
alpaca load <TAB>          # List all available presets and models
alpaca load p:<TAB>        # Complete preset names
alpaca load h:<TAB>        # Complete downloaded model identifiers
alpaca load f:<TAB>        # No completion (type full path manually)
```

**Design Decision:** Instead of showing bare prefixes (`p:`, `h:`, `f:`), we show actual items directly. This solves the shell auto-spacing issue where `p: ` (with trailing space) would prevent further completion. Users can still type `p:` or `h:` to filter, or select items directly from the initial list.

## Implementation Plan

### 1. Add kongplete Dependency

```bash
go get github.com/willabides/kongplete@v0.4.0
```

### 2. Integrate kongplete in main.go

```go
package main

import (
    "github.com/alecthomas/kong"
    "github.com/willabides/kongplete"
)

type CLI struct {
    // Existing commands...
    Start   StartCmd   `cmd:"" help:"Start the daemon"`
    Stop    StopCmd    `cmd:"" help:"Stop the daemon"`
    Status  StatusCmd  `cmd:"" help:"Show current status"`
    Load    LoadCmd    `cmd:"" help:"Load a preset, model, or file"`
    Unload  UnloadCmd  `cmd:"" help:"Stop the currently running model"`
    Logs    LogsCmd    `cmd:"" help:"Show logs (daemon or server)"`
    List    ListCmd    `cmd:"" name:"ls" help:"List presets and models"`
    Show    ShowCmd    `cmd:"" help:"Show details of a preset or model"`
    Remove  RemoveCmd  `cmd:"" name:"rm" help:"Remove a preset or model"`
    Pull    PullCmd    `cmd:"" help:"Download a model"`
    New     NewCmd     `cmd:"" help:"Create a new preset interactively"`
    Version VersionCmd `cmd:"" help:"Show version"`

    // Completion command
    InstallCompletions kongplete.InstallCompletions `cmd:"" help:"Install shell completions"`
}

func main() {
    cli := CLI{}
    parser, err := kong.New(&cli,
        kong.Name("alpaca"),
        kong.Description("Lightweight llama-server wrapper"),
        kong.UsageOnError(),
        kong.ConfigureHelp(kong.HelpOptions{
            Compact: true,
        }),
    )
    if err != nil {
        panic(err)
    }

    // Add completion support with different predictors per command
    kongplete.Complete(parser,
        kongplete.WithPredictor("show-identifier", newShowIdentifierPredictor()),
        kongplete.WithPredictor("rm-identifier", newRmIdentifierPredictor()),
        kongplete.WithPredictor("load-identifier", newLoadIdentifierPredictor()),
    )

    ctx, err := parser.Parse(os.Args[1:])
    if err != nil {
        parser.FatalIfErrorf(err)
    }

    err = ctx.Run()
    if err != nil {
        // ... existing error handling
    }
}
```

### 3. Update Command Structs with Completion Predictor

```go
type LoadCmd struct {
    Identifier string `arg:"" help:"Identifier (p:preset, h:org/repo:quant, or f:/path/to/file)" predictor:"load-identifier"`
}

type ShowCmd struct {
    Identifier string `arg:"" help:"Show details (p:name or h:org/repo:quant)" predictor:"show-identifier"`
}

type RemoveCmd struct {
    Identifier string `arg:"" help:"Remove target (p:name or h:org/repo:quant)" predictor:"rm-identifier"`
}
```

### 4. Implement Custom Identifier Predictors

Create `cmd/alpaca/completion.go`:

```go
package main

import (
    "context"
    "strings"

    "github.com/d2verb/alpaca/internal/model"
    "github.com/d2verb/alpaca/internal/preset"
    "github.com/posener/complete"
)

// newShowIdentifierPredictor returns a predictor for 'show' command.
// Supports: p:preset-name, h:org/repo:quant
func newShowIdentifierPredictor() complete.Predictor {
    return newIdentifierPredictor([]string{"p:", "h:"})
}

// newRmIdentifierPredictor returns a predictor for 'rm' command.
// Supports: p:preset-name, h:org/repo:quant
func newRmIdentifierPredictor() complete.Predictor {
    return newIdentifierPredictor([]string{"p:", "h:"})
}

// newLoadIdentifierPredictor returns a predictor for 'load' command.
// Supports: p:preset-name, h:org/repo:quant, f:/path/to/file
func newLoadIdentifierPredictor() complete.Predictor {
    return newIdentifierPredictor([]string{"p:", "h:", "f:"})
}

// identifierPredictor implements complete.Predictor for identifier completion.
type identifierPredictor struct {
    validPrefixes []string
}

// newIdentifierPredictor returns a predictor that completes identifiers based on prefix.
// validPrefixes determines which prefixes are valid for this command.
func newIdentifierPredictor(validPrefixes []string) complete.Predictor {
    return &identifierPredictor{validPrefixes: validPrefixes}
}

// Predict implements complete.Predictor interface.
func (p *identifierPredictor) Predict(args complete.Args) []string {
    // Get the current value being completed
    value := args.Last

    // Get paths early to avoid errors during completion
    paths, err := getPaths()
    if err != nil {
        return nil
    }

    // Note: Using context.Background() here because complete.Predictor interface
    // doesn't provide context. This is acceptable for completion use case where
    // operations are expected to be fast (<100ms).
    ctx := context.Background()

    // Determine completion based on prefix
    switch {
    case value == "":
        // No input yet - show actual items from all valid prefixes
        // This avoids the auto-spacing issue where "p: " would prevent further completion
        var results []string
        for _, prefix := range p.validPrefixes {
            switch prefix {
            case "p:":
                results = append(results, completePresets(ctx, paths.Presets, "p:")...)
            case "h:":
                results = append(results, completeModels(ctx, paths.Models, "h:")...)
            case "f:":
                // f: prefix doesn't have completion
            }
        }
        return results

    case strings.HasPrefix(value, "p:"):
        // Preset completion: p:name
        return completePresets(ctx, paths.Presets, value)

    case strings.HasPrefix(value, "h:"):
        // HuggingFace model completion: h:org/repo:quant
        return completeModels(ctx, paths.Models, value)

    case strings.HasPrefix(value, "f:"):
        // File path completion - no completion support
        // Users can manually type the full path
        return nil

    default:
        // Invalid input - show actual items from valid prefixes
        var results []string
        for _, prefix := range p.validPrefixes {
            switch prefix {
            case "p:":
                results = append(results, completePresets(ctx, paths.Presets, prefix)...)
            case "h:":
                results = append(results, completeModels(ctx, paths.Models, prefix)...)
            }
        }
        return results
    }
}

// completePresets returns preset name completions.
func completePresets(ctx context.Context, presetsDir, partial string) []string {
    loader := preset.NewLoader(presetsDir)
    names, err := loader.List()
    if err != nil {
        return nil
    }

    // Add "p:" prefix to each name
    results := make([]string, 0, len(names))
    for _, name := range names {
        completion := "p:" + name
        if strings.HasPrefix(completion, partial) {
            results = append(results, completion)
        }
    }
    return results
}

// completeModels returns downloaded model identifier completions.
func completeModels(ctx context.Context, modelsDir, partial string) []string {
    modelMgr := model.NewManager(modelsDir)
    entries, err := modelMgr.List(ctx)
    if err != nil {
        return nil
    }

    // Build h:org/repo:quant format
    results := make([]string, 0, len(entries))
    for _, entry := range entries {
        completion := "h:" + entry.Repo + ":" + entry.Quant
        if strings.HasPrefix(completion, partial) {
            results = append(results, completion)
        }
    }
    return results
}
```

## User Experience

### Installation

The `install-completions` command outputs shell completion script to stdout. You need to add it to your shell configuration file.

**Bash:**
```bash
# Add to ~/.bashrc
alpaca install-completions >> ~/.bashrc
source ~/.bashrc  # Or restart your terminal
```

**Zsh:**
```bash
# Add to ~/.zshrc
alpaca install-completions >> ~/.zshrc
source ~/.zshrc   # Or restart your terminal
```

**Fish:**
```bash
# Create completion file
mkdir -p ~/.config/fish/completions
alpaca install-completions > ~/.config/fish/completions/alpaca.fish
# Fish automatically loads completions, no restart needed
```

**Important:** After installing completions, you must either:
- Restart your terminal, or
- Source your shell configuration file (shown above for Bash/Zsh)

### Uninstallation

To remove completions, manually delete the lines added to your shell configuration file, or use:

```bash
alpaca install-completions --uninstall
```

This outputs instructions for removing the completion setup.

### Usage Examples

**Show command (p: and h: only):**
```bash
$ alpaca show <TAB>
# Shows all available presets and models directly
p:codellama-7b  p:gemma-2b  p:llama3-8b
h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
h:unsloth/gemma-2-2b-it-bnb-4bit:Q4_K_M

$ alpaca show p:<TAB>
p:codellama-7b  p:gemma-2b  p:llama3-8b

$ alpaca show p:code<TAB>
p:codellama-7b

$ alpaca show h:<TAB>
h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
h:unsloth/gemma-2-2b-it-bnb-4bit:Q4_K_M
```

**Remove command (p: and h: only):**
```bash
$ alpaca rm <TAB>
# Shows all available presets and models directly
p:codellama-7b  p:gemma-2b  p:llama3-8b
h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
h:unsloth/gemma-2-2b-it-bnb-4bit:Q4_K_M

$ alpaca rm p:<TAB>
p:codellama-7b  p:gemma-2b  p:llama3-8b

$ alpaca rm h:<TAB>
h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
h:unsloth/gemma-2-2b-it-bnb-4bit:Q4_K_M
```

**Load command (p:, h:, and f:):**
```bash
$ alpaca load <TAB>
# Shows all available presets and models directly
# (f: is valid but has no completions)
p:codellama-7b  p:gemma-2b  p:llama3-8b
h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
h:unsloth/gemma-2-2b-it-bnb-4bit:Q4_K_M

$ alpaca load p:<TAB>
p:codellama-7b  p:gemma-2b  p:llama3-8b

$ alpaca load h:<TAB>
h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
h:unsloth/gemma-2-2b-it-bnb-4bit:Q4_K_M

$ alpaca load f:<TAB>
# No completions (type full path manually)
# Example: alpaca load f:/path/to/model.gguf
```

## Progressive Completion Strategy

### Phase 1: Static Completion (Future Enhancement)
For HuggingFace models not yet downloaded, we could add static completion from popular models list. This would require:

1. Maintain curated list of popular models in `internal/completion/popular.go`
2. Enhance `completeModels()` to merge downloaded + popular models
3. Mark undownloaded models with visual indicator (if shell supports)

Example:
```go
func completeModels(modelsDir, partial string) []string {
    // Get downloaded models
    downloaded := getDownloadedModels(modelsDir)

    // Get popular models (static list)
    popular := getPopularModels()

    // Merge and deduplicate
    return mergeCompletions(downloaded, popular, partial)
}
```

This is deferred to avoid bloating the binary and maintaining model lists.

## Technical Notes

### Why kongplete?

- **Official kong support**: Maintained by kong ecosystem
- **Shell-agnostic**: Single implementation for bash/zsh/fish
- **Custom predictors**: Dynamic completion based on filesystem state
- **Zero runtime dependencies**: Completion scripts are standalone

### Architecture: kongplete + posener/complete

The implementation uses two packages:

1. **kongplete (willabides/kongplete)**: Kong integration layer
   - Provides `Complete()` function to register predictors with kong parser
   - Handles shell detection and script generation via `InstallCompletions`

2. **posener/complete**: Actual completion engine
   - Provides `Predictor` interface: `Predict(Args) []string`
   - kongplete internally uses this package
   - Our custom predictors implement `complete.Predictor`

This separation is intentional: `kongplete.WithPredictor()` expects `complete.Predictor`, so we implement the interface from `posener/complete` while using `kongplete` for kong integration.

### Performance Considerations

- Completion functions must be fast (<100ms)
- Preset listing: O(n) directory scan, typically <10 files
- Model listing: O(n) metadata reads, typically <20 files
- Both operations are acceptable for interactive use

### Limitations

1. **Partial path completion**: `h:TheBloke/Code<TAB>` won't complete to full repo name (requires fuzzy matching)
2. **Quant inference**: Cannot suggest quant values for undownloaded models (requires HF API call)
3. **File path completion**: Not implemented for `f:` prefix. Users must type the full path manually after `f:`

These limitations are acceptable for MVP. The primary use cases (preset and model completion) are fully supported.

## Testing

### Unit Tests

While full shell integration testing is complex, we should add unit tests for the helper functions.

Create `cmd/alpaca/completion_test.go`:

```go
package main

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestCompletePresets(t *testing.T) {
    // Arrange: Create temp directory with test presets
    tmpDir := t.TempDir()
    presets := []string{"codellama", "gemma-2b", "llama3"}
    for _, name := range presets {
        err := os.WriteFile(filepath.Join(tmpDir, name+".yaml"), []byte("model: test"), 0644)
        if err != nil {
            t.Fatalf("failed to create preset: %v", err)
        }
    }

    tests := []struct {
        name     string
        partial  string
        expected int
    }{
        {"no filter", "p:", 3},
        {"partial match", "p:code", 1},
        {"no match", "p:xyz", 0},
        {"empty input", "", 3},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act
            ctx := context.Background()
            results := completePresets(ctx, tmpDir, tt.partial)

            // Assert
            if len(results) != tt.expected {
                t.Errorf("expected %d results, got %d: %v", tt.expected, len(results), results)
            }

            // Verify all results have p: prefix
            for _, r := range results {
                if len(r) < 2 || r[:2] != "p:" {
                    t.Errorf("result missing p: prefix: %s", r)
                }
            }
        })
    }
}

func TestCompleteModels(t *testing.T) {
    // Arrange: Create temp directory with test metadata
    tmpDir := t.TempDir()
    metadataContent := `{
        "models": [
            {
                "repo": "TheBloke/CodeLlama-7B-GGUF",
                "quant": "Q4_K_M",
                "filename": "codellama-7b.Q4_K_M.gguf",
                "size": 4368438272,
                "downloaded_at": "2024-01-01T00:00:00Z"
            },
            {
                "repo": "unsloth/gemma-2-2b-it-bnb-4bit",
                "quant": "Q4_K_M",
                "filename": "gemma-2-2b.Q4_K_M.gguf",
                "size": 1610612736,
                "downloaded_at": "2024-01-01T00:00:00Z"
            }
        ]
    }`
    err := os.WriteFile(filepath.Join(tmpDir, ".metadata.json"), []byte(metadataContent), 0644)
    if err != nil {
        t.Fatalf("failed to create metadata: %v", err)
    }

    tests := []struct {
        name     string
        partial  string
        expected int
    }{
        {"no filter", "h:", 2},
        {"partial match", "h:TheBloke", 1},
        {"no match", "h:xyz", 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Act
            ctx := context.Background()
            results := completeModels(ctx, tmpDir, tt.partial)

            // Assert
            if len(results) != tt.expected {
                t.Errorf("expected %d results, got %d: %v", tt.expected, len(results), results)
            }

            // Verify all results have h: prefix and include quant
            for _, r := range results {
                if len(r) < 2 || r[:2] != "h:" {
                    t.Errorf("result missing h: prefix: %s", r)
                }
            }
        })
    }
}

func TestCompleteModels_EmptyMetadata(t *testing.T) {
    // Arrange: Empty directory (no metadata file)
    tmpDir := t.TempDir()

    // Act
    ctx := context.Background()
    results := completeModels(ctx, tmpDir, "h:")

    // Assert: Should return empty slice, not nil
    if results == nil {
        t.Error("expected empty slice, got nil")
    }
    if len(results) != 0 {
        t.Errorf("expected 0 results, got %d", len(results))
    }
}
```

These tests verify:
- Preset and model completion logic
- Prefix filtering
- Error handling (missing metadata)
- AAA pattern structure

Note: These tests cover the helper functions only. Full shell integration requires manual testing.

### Manual Testing

Manual testing checklist:

```bash
# Test show command (p: and h: only)
alpaca show <TAB>           # Should list all presets and models
alpaca show p:<TAB>         # Should list presets only
alpaca show h:<TAB>         # Should list models only
alpaca show f:<TAB>         # Should list all presets and models (f: is invalid)

# Test rm command (p: and h: only)
alpaca rm <TAB>             # Should list all presets and models
alpaca rm p:<TAB>           # Should list presets only
alpaca rm h:<TAB>           # Should list models only
alpaca rm f:<TAB>           # Should list all presets and models (f: is invalid)

# Test load command (p:, h:, and f:)
alpaca load <TAB>           # Should list all presets and models
alpaca load p:<TAB>         # Should list presets only
alpaca load h:<TAB>         # Should list models only
alpaca load f:<TAB>         # Should show no completions (type path manually)

# Test partial completion
alpaca show p:code<TAB>     # Should complete matching presets
alpaca rm h:TheBloke<TAB>   # Should complete matching models
```

No automated tests for completion (kongplete handles shell integration).

## Documentation Updates

Update `README.md`:

```markdown
## Shell Completion

Install completions for your shell by adding the output to your shell configuration:

```bash
# Bash (add to ~/.bashrc)
alpaca install-completions >> ~/.bashrc
source ~/.bashrc

# Zsh (add to ~/.zshrc)
alpaca install-completions >> ~/.zshrc
source ~/.zshrc

# Fish
mkdir -p ~/.config/fish/completions
alpaca install-completions > ~/.config/fish/completions/alpaca.fish
```

Completions support:
- Direct item completion: `alpaca show <TAB>` lists all presets and models
- Prefix filtering: Type `p:` or `h:` to filter by type
- Preset names: `alpaca show p:<TAB>`, `alpaca rm p:<TAB>`, `alpaca load p:<TAB>`
- Downloaded models: `alpaca show h:<TAB>`, `alpaca rm h:<TAB>`, `alpaca load h:<TAB>`
- Command flags: `alpaca start --<TAB>`

Note: File path completion (`f:`) is not implemented. Type the full path manually after `f:`.
```

## References

- [kongplete Documentation](https://github.com/willabides/kongplete)
- [kong Shell Completion](https://github.com/alecthomas/kong#shell-completion)
