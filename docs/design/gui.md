# GUI Design

## Overview

The GUI is a macOS menu bar app built with AppKit. It provides quick access to status and common operations without leaving the current context.

## Menu Bar Icon

Location: macOS menu bar (top-right area)

Icon: SF Symbol `brain`
- Represents AI/ML processing
- Monochrome, adapts to light/dark mode

## Menu States

### State 1: Daemon Not Running

```
┌─────────────────────────────────┐
│  ○ Daemon not running           │  ← Gray indicator
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Gray status indicator
- Action items hidden
- User must start daemon via CLI: `alpaca start`

### State 2: Daemon Running, No Model Loaded (Idle)

```
┌─────────────────────────────────┐
│  ● Idle                         │  ← Yellow indicator
│  No model loaded                │
├─────────────────────────────────┤
│  ▶ Load Model...                │  → Model/Preset submenu
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Yellow status indicator
- "Load Model..." opens model/preset selection submenu

### State 3: Model Running

```
┌─────────────────────────────────┐
│  ● Running                      │  ← Green indicator
│  codellama-7b-q4                │  ← Current model/preset
│  http://localhost:8080          │  ← Endpoint (monospace)
├─────────────────────────────────┤
│  🌐 Open in Browser             │
├─────────────────────────────────┤
│  ▶ Switch Model...              │  → Model/Preset submenu
│  ■ Stop                         │
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Green status indicator
- Current model/preset name and endpoint displayed
- "Open in Browser" opens the llama-server web UI
- "Switch Model..." for quick switching
- "Stop" to unload model

### State 4: Loading Model

```
┌─────────────────────────────────┐
│  ◐ Loading...                   │  ← Blue indicator (static)
│  mistral-7b-q4                  │  ← Target model/preset
├─────────────────────────────────┤
│  ✕ Cancel                       │
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Blue half-circle indicator (static, not animated)
- Shows target model/preset name
- Cancel option to abort loading

### Error Display

When an error occurs, an additional line appears below the status:

```
│  ⚠ Connection failed            │  ← Red text
```

## Model/Preset Submenu

When "Load Model..." or "Switch Model..." is clicked:

```
┌─────────────────────────────────┐
│  Downloaded Models              │  ← Section header (disabled)
│  codellama-7b-q4           ✓    │  ← Currently loaded (if any)
│  mistral-7b-q4                  │
├─────────────────────────────────┤
│  Presets                        │  ← Section header (disabled)
│  deepseek-coder                 │
│  llama3-8b                      │
├─────────────────────────────────┤
│  No models or presets available │  ← Shown when both empty
└─────────────────────────────────┘
```

- Two sections: "Downloaded Models" and "Presets"
- Checkmark on currently loaded item
- Click to switch immediately (no confirmation)
- Tooltip shows identifier and size for models

## Interaction Behaviors

### Model Switching

1. User clicks model/preset in submenu
2. UI immediately shows "Loading..." state
3. Daemon gracefully stops current llama-server
4. Daemon starts llama-server with new model/preset
5. UI updates to "Running" when ready

No confirmation dialog - switch is immediate.

### Open in Browser

In "Model Running" state:
- Click opens the llama-server endpoint in default browser
- Provides access to llama-server's built-in web UI
- Only visible when a model is actively running

### Quit Alpaca

- Close GUI application
- Note: Does NOT stop the daemon or running models (daemon runs independently)
