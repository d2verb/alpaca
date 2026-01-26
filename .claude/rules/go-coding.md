---
paths: "**/*.go"
---

# Go Coding Rules

## Error Handling (REQUIRED)

ALWAYS wrap errors with context:

```go
// WRONG: Bare error return
func loadPreset(name string) (*Preset, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
}

// CORRECT: Wrapped with context
func loadPreset(name string) (*Preset, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("load preset %s: %w", name, err)
    }
}
```

## Context Propagation

ALWAYS pass context.Context as first parameter:

```go
func (d *Daemon) RunModel(ctx context.Context, preset string) error {
    // ...
}
```

## Naming

- Package names: singular, short (`preset` not `presets`)
- Variables: short in small scopes (`p` for preset in a loop)
- Exported: descriptive (`LoadPreset` not `Load`)
