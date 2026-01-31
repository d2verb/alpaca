# CLI Output Design System

Unified terminal output design system for Alpaca CLI.

This document defines the visual language, color palette, layout rules, and output components for all Alpaca CLI commands. The design system ensures consistent, readable, and professional terminal output across all user interactions.

## Design Principles

1. **Semantic Colors**: Use colors for meaning, not aesthetics - every color has a purpose
2. **Visual Hierarchy**: Header > Data > Metadata in prominence - guide the eye to important information
3. **Consistency**: All output goes through the ui package - no direct stdout/stderr writes in commands
4. **Breathing Room**: Appropriate spacing for readability - balance density with clarity
5. **Terminal-First**: Optimized for monospace fonts and 80-column terminals

## Color Palette

### Semantic Colors

```go
// Primary: Identifiers, names, and primary data
Primary = color.New(color.FgCyan, color.Bold).SprintFunc()

// Secondary: Supplementary data (quant, type, etc.)
Secondary = color.New(color.FgMagenta).SprintFunc()

// Link: Paths, URLs (clickable impression)
Link = color.New(color.FgBlue, color.Underline).SprintFunc()

// Status colors
Success = color.New(color.FgGreen).SprintFunc()
Error = color.New(color.FgRed).SprintFunc()
Warning = color.New(color.FgYellow).SprintFunc()
Info = color.New(color.FgCyan).SprintFunc()

// Muted: Supplementary info (size, timestamps, etc.) - using normal color for readability
Muted = func(s string) string { return s }

// Emphasis (headers, labels)
Heading = color.New(color.FgWhite, color.Bold).SprintFunc()
Label = func(s string) string { return s } // Normal color for readability
```

### Usage Guidelines

| Element | Color | Example |
|---------|-------|---------|
| Preset name | Primary (Cyan+Bold) | `codellama-7b` |
| Model repository | Primary (Cyan+Bold) | `TheBloke/CodeLlama-7B-GGUF` |
| Quant | Secondary (Magenta) | `Q4_K_M` |
| File path | Link (Blue+Underline) | `/path/to/model.gguf` |
| URL/Endpoint | Link (Blue+Underline) | `http://localhost:8080` |
| Size, timestamps | Normal (no color) | `4.1 GB`, `2024-01-15` |
| Section headers | Heading (White+Bold) | `Presets`, `Models` |
| Key labels | Normal (no color) | `Name:`, `Model:` |

## Output Components

### 1. Section Header

Section dividers for list outputs.

```
📦 Presets
──────────
```

Implementation:
```go
// Header for lists (with divider)
func PrintSectionHeader(icon, title string) {
    fmt.Fprintf(Output, "\n%s %s\n", icon, Heading(title))
    fmt.Fprintln(Output, Muted(strings.Repeat("─", len(title)+2)))
}

// Header for detail views (no divider, title only)
func PrintDetailHeader(icon, title, identifier string) {
    fmt.Fprintf(Output, "\n%s %s: %s\n", icon, Heading(title), identifier)
}
```

### 2. List Output

Preset and model listings.

#### Preset List

```
📦 Presets
──────────
  p:codellama-7b
  p:mistral-7b
  p:deepseek-coder
```

Implementation:
```go
func PrintPresetList(presets []string) {
    if len(presets) == 0 {
        PrintEmptyState("No presets available", "alpaca new")
        return
    }

    PrintSectionHeader("📦", "Presets")
    for _, name := range presets {
        // Display with p: prefix (matches command input)
        fmt.Fprintf(Output, "  %s%s\n", Primary("p:"), Primary(name))
    }
}
```

#### Model List

```
🤖 Models
─────────
  h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
    4.1 GB · Downloaded 2024-01-15
  h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
    4.8 GB · Downloaded 2024-01-14
```

Implementation:
```go
func PrintModelList(models []ModelInfo) {
    if len(models) == 0 {
        PrintEmptyState("No models downloaded", "alpaca pull h:org/repo:quant")
        return
    }

    PrintSectionHeader("🤖", "Models")
    for _, m := range models {
        // Display in full h:repo:quant format (matches command input)
        fmt.Fprintf(Output, "  %s%s:%s\n",
            Primary("h:"),
            Primary(m.Repo),
            Secondary(m.Quant),
        )
        fmt.Fprintf(Output, "    %s · Downloaded %s\n",
            m.SizeString,
            m.DownloadedAt,
        )
    }
}
```

### 3. Detail Display (Key-Value)

Preset and model details.

#### Preset Details

```
📦 Preset: p:codellama-7b
  Model          /Users/.../models/codellama-7b.gguf
  Context Size   8192
  GPU Layers     32
  Threads        8
  Endpoint       http://localhost:8080
  Extra Args     --n-predict 512 --temperature 0.7
```

Implementation:
```go
func PrintPresetDetails(p PresetDetails) {
    // Display with p: prefix
    identifier := fmt.Sprintf("%s:%s", Label("p"), Primary(p.Name))
    PrintDetailHeader("📦", "Preset", identifier)

    PrintKeyValue("Model", Link(p.Model))
    if p.ContextSize > 0 {
        PrintKeyValue("Context Size", fmt.Sprintf("%d", p.ContextSize))
    }
    if p.GPULayers != 0 {
        PrintKeyValue("GPU Layers", fmt.Sprintf("%d", p.GPULayers))
    }
    if p.Threads > 0 {
        PrintKeyValue("Threads", fmt.Sprintf("%d", p.Threads))
    }
    PrintKeyValue("Endpoint", Link(fmt.Sprintf("%s:%d", p.Host, p.Port)))
    if len(p.ExtraArgs) > 0 {
        PrintKeyValue("Extra Args", strings.Join(p.ExtraArgs, " "))
    }
}

func PrintKeyValue(key, value string) {
    fmt.Fprintf(Output, "  %-14s %s\n", key, value)
}
```

#### Model Details

```
🤖 Model: h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
  Filename       codellama-7b.Q4_K_M.gguf
  Size           4.1 GB
  Downloaded     2024-01-15 10:30:00
  Path           /Users/.../models/codellama-7b.gguf
  Status         ✓ Ready
```

Implementation:
```go
func PrintModelDetails(m ModelDetails) {
    // Display in full h:repo:quant format
    identifier := fmt.Sprintf("%s:%s:%s",
        Label("h"),
        Primary(m.Repo),
        Secondary(m.Quant),
    )
    PrintDetailHeader("🤖", "Model", identifier)

    PrintKeyValue("Filename", m.Filename)
    PrintKeyValue("Size", m.Size)
    PrintKeyValue("Downloaded", Muted(m.DownloadedAt))
    PrintKeyValue("Path", Link(m.Path))
    PrintKeyValue("Status", Success("✓ Ready"))
}
```

### 4. Status Display

Daemon status display.

```
🚀 Status
  State      ● Running
  Preset     p:codellama-7b
  Endpoint   http://localhost:8080
  Logs       /Users/.../logs/daemon.log
```

Implementation:
```go
func PrintStatus(state, preset, endpoint, logPath string) {
    fmt.Fprintf(Output, "\n🚀 %s\n", Heading("Status"))

    PrintKeyValue("State", StatusBadge(state))
    if preset != "" {
        // Display with p: prefix
        PrintKeyValue("Preset", fmt.Sprintf("%s:%s", Label("p"), Primary(preset)))
    }
    if endpoint != "" {
        PrintKeyValue("Endpoint", Link(endpoint))
    }
    PrintKeyValue("Logs", Muted(logPath))
}
```

### 5. Messages (Feedback)

Operation result messages.

```
✓ Model ready at http://localhost:8080
ℹ Model not found. Downloading...
⚠ Daemon did not stop gracefully, forcing...
✗ Preset 'foo' not found
```

Implementation:
```go
func PrintSuccess(message string) {
    fmt.Fprintf(Output, "%s %s\n", Success("✓"), message)
}

func PrintError(message string) {
    fmt.Fprintf(Output, "%s %s\n", Error("✗"), message)
}

func PrintWarning(message string) {
    fmt.Fprintf(Output, "%s %s\n", Warning("⚠"), message)
}

func PrintInfo(message string) {
    fmt.Fprintf(Output, "%s %s\n", Info("ℹ"), message)
}
```

### 6. Empty State

Display when no data exists.

```
No presets available.

  Create one:  alpaca new
```

Implementation:
```go
func PrintEmptyState(message, suggestion string) {
    fmt.Fprintf(Output, "\n%s\n\n", Muted(message))
    if suggestion != "" {
        fmt.Fprintf(Output, "  %s  %s\n\n", Label("Create one:"), Info(suggestion))
    }
}
```


## Icon System

| Purpose | Icon | Color |
|---------|------|-------|
| Preset | 📦 | - |
| Model | 🤖 | - |
| Status | 🚀 | - |
| Success | ✓ | Green |
| Error | ✗ | Red |
| Warning | ⚠ | Yellow |
| Info | ℹ | Cyan |
| Running | ● | Green |
| Loading | ◐ | Yellow |
| Stopped | ○ | Red |

## Layout Rules

### Divider Usage Rules

```
✅ Use: List outputs only (multiple items)
   - alpaca ls preset/model listings

❌ Don't use: Detail displays and status
   - alpaca show (detail view)
   - alpaca status (status view)
```

### Spacing

```
- Before first section: No blank line
- Between sections: 1 blank line (added by caller)
- After section: Divider only (no blank line)
- Between list items (models): No blank line (compact)
- Detail display: Compact from title down (no blank lines)
- Between key-values: No blank lines (compact)
- After messages: No blank lines
```

### Indentation

```
- Content within sections: 2 spaces
- Nested data (model details, etc.): 4 spaces
- Key-value alignment: Keys left-aligned at 14 characters
```

### Key-Value Alignment

```go
// Keys aligned to 14 characters, values to the right
  Model          /path/to/model.gguf
  Context Size   8192
  GPU Layers     32
```

## Usage Examples

### alpaca ls

```
📦 Presets
──────────
  p:codellama-7b
  p:mistral-7b
  p:deepseek-coder

🤖 Models
─────────
  h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
    4.1 GB · Downloaded 2024-01-15
  h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
    4.8 GB · Downloaded 2024-01-14
```

Note: No leading blank line, single blank line between sections.

### alpaca status

```
🚀 Status
  State      ● Running
  Preset     p:codellama-7b
  Endpoint   http://localhost:8080
  Logs       /Users/user/.alpaca/logs/daemon.log
```

Note: No leading blank line.

### alpaca show p:codellama-7b

```
📦 Preset: p:codellama-7b
  Model          /Users/user/.alpaca/models/codellama-7b.gguf
  Context Size   8192
  GPU Layers     32
  Threads        8
  Endpoint       http://localhost:8080
  Extra Args     --n-predict 512 --temperature 0.7
```

### alpaca load h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M

```
ℹ Loading h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M...
✓ Model ready at http://localhost:8080
```

### alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M

```
⏳ Downloading model... 45% (2.1 GB / 4.6 GB)
✓ Model downloaded successfully
```

### alpaca ls (empty state)

```
No presets available.

  Create one:  alpaca new

No models downloaded.

  Download one:  alpaca pull h:org/repo:quant
```

## Before/After Comparison

### alpaca ls

**Before:**
```
Available presets:
  p:codellama-7b
  p:mistral-7b

Downloaded models:
  h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M (4.1 GB)
```

**After:**
```
📦 Presets
──────────
  p:codellama-7b
  p:mistral-7b

🤖 Models
─────────
  h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
    4.1 GB · Downloaded 2024-01-15
```

**Improvements:**
- ✅ Add icons to sections (📦/🤖) for easy identification
- ✅ Keep prefixes (`p:`/`h:`) for command input consistency and copy-paste
- ✅ Split model details into 2 lines for readability
- ✅ Add blank lines between sections
- ✅ Dividers clarify list boundaries

### alpaca show p:codellama-7b

**Before:**
```
Name: codellama-7b
Model: /path/to/model.gguf
Context Size: 8192
GPU Layers: 32
Endpoint: http://localhost:8080:8080
Extra Args: [--n-predict 512]
```

**After:**
```
📦 Preset: p:codellama-7b
  Model          /Users/user/.alpaca/models/codellama-7b.gguf
  Context Size   8192
  GPU Layers     32
  Threads        8
  Endpoint       http://localhost:8080
  Extra Args     --n-predict 512 --temperature 0.7
```

**Improvements:**
- ✅ Prominent title (`📦 Preset: p:codellama-7b`)
- ✅ Prefix matches command input
- ✅ Left-align labels (14 characters)
- ✅ Compact display (no blank lines)
- ✅ Underline paths and URLs (clickable impression)

## Migration Plan

### Phase 1: Redefine Color Palette

- Update color definitions in `internal/ui/ui.go` to semantic names
- Keep existing colors (Green, Red, etc.) for backward compatibility
- Add new colors (Primary, Secondary, Link, etc.)

### Phase 2: Add New Components

- Add `PrintSectionHeader()`
- Add `PrintKeyValue()`
- Add `PrintEmptyState()`

### Phase 3: Refactor Existing Components

- Migrate `PrintPresetList()` to new format
- Migrate `PrintModelList()` to new format
- Migrate `PrintPresetDetails()` to new format
- Migrate `PrintModelDetails()` to new format
- Migrate `PrintStatus()` to new format

### Phase 4: Migrate Commands

- Unify all commands (`cmd_*.go`) through ui package
- Fix direct `fmt.Printf` usage (e.g., `cmd_version.go`)

### Phase 5: Update Tests

- Update `internal/ui/ui_test.go` for new formats
- Consider adding snapshot tests

## Testing Strategy

### Visual Testing

Manually execute each command to verify output:

```bash
alpaca ls
alpaca status
alpaca show p:codellama-7b
alpaca show h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```

### Unit Testing

Verify with color codes stripped:

```go
func TestPrintModelList(t *testing.T) {
    // ... (existing tests)

    // Verify new format
    if !strings.Contains(stripColors(output), "🤖 Models") {
        t.Error("Output should contain section header with icon")
    }
    if !strings.Contains(stripColors(output), "Q4_K_M") {
        t.Error("Output should contain formatted model details")
    }
}
```

## NO-GO (What Not To Do)

- ✗ Excessive animations (keep CLI simple)
- ✗ ASCII Art (alpaca logo, etc.)
- ✗ Excessive emojis (max 1 per section)
- ✗ Custom table libraries (standard output is sufficient)
- ✗ Color customization (user settings add complexity)

## Reference Implementations

Excellent CLI output examples from similar tools:

- **gh**: GitHub CLI (section headers, key-value, status display)
- **docker**: Docker CLI (tables, progress)
- **ollama**: Ollama CLI (model lists, progress)
- **kubectl**: Kubernetes CLI (resource display, status)

## Scope

This design system covers:
- ✅ Terminal output formatting (stdout)
- ✅ Color usage and semantic meaning
- ✅ Layout and spacing rules
- ✅ Text alignment and typography
- ✅ Icons and symbols

This design system does NOT cover:
- ❌ Error messages (stderr) - handled separately
- ❌ Interactive prompts - handled by individual commands
- ❌ GUI applications
- ❌ Web interfaces
