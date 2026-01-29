# GUI Design

## Overview

The GUI is a macOS menu bar app built with SwiftUI. It provides quick access to status and common operations without leaving the current context.

## Menu Bar Icon

Location: macOS menu bar (top-right area)

Icon: SF Symbol `brain`
- Represents AI/ML processing
- Monochrome, adapts to light/dark mode
- Could indicate status with color overlay in future

## Popover States

### State 1: Daemon Not Running

```
┌─────────────────────────────────┐
│  ○ Daemon not running           │
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Gray status indicator
- Minimal menu with only quit option
- User must start daemon via CLI: `alpaca start`

### State 2: Daemon Running, No Model Loaded (Idle)

```
┌─────────────────────────────────┐
│  ○ Idle                         │  ← Yellow indicator
│  No model loaded                │
├─────────────────────────────────┤
│  ▶ Load Model...                │  → Preset submenu
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Yellow status indicator
- "Load Model..." opens preset selection submenu
- Quit option

### State 3: Model Running

```
┌─────────────────────────────────┐
│  ● Running                      │  ← Green indicator
│  codellama-7b-q4                │  ← Current preset name
│  localhost:8080                 │  ← Endpoint
├─────────────────────────────────┤
│  🌐 Open in Browser             │  ← Opens endpoint in browser
├─────────────────────────────────┤
│  ▶ Switch Model...              │  → Preset submenu
│  ■ Stop                         │
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Green status indicator
- Current preset name and endpoint displayed
- "Open in Browser" opens the llama-server web UI
- "Switch Model..." for quick switching
- "Stop" to unload model
- Quit option

### State 4: Loading/Switching Model

```
┌─────────────────────────────────┐
│  ◐ Loading...                   │  ← Animated indicator
│  mistral-7b-q4                  │  ← Target preset name
├─────────────────────────────────┤
│  ✕ Cancel                       │
├─────────────────────────────────┤
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Animated loading indicator
- Shows target preset name
- Cancel option to abort loading
- Quit option

## Preset Submenu

When "Load Model..." or "Switch Model..." is clicked:

```
┌─────────────────────────────────┐
│  codellama-7b-q4           ✓    │  ← Currently loaded (if any)
│  mistral-7b-q4                  │
│  deepseek-coder-6.7b            │
│  llama3-8b-q4                   │
└─────────────────────────────────┘
```

- Lists all available presets
- Checkmark on currently loaded preset
- Click to switch immediately (no confirmation)

## Interaction Behaviors

### Model Switching

1. User clicks preset in submenu
2. UI immediately shows "Loading..." state
3. Daemon gracefully stops current llama-server
4. Daemon starts llama-server with new preset
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
