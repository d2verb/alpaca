# GUI Design

## Overview

The GUI is a macOS menu bar app built with SwiftUI. It provides quick access to status and common operations without leaving the current context.

## Menu Bar Icon

Location: macOS menu bar (top-right area)

Icon options:
- Alpaca emoji or simple alpaca icon
- Could indicate status with color overlay

## Popover States

### State 1: Daemon Not Running

```
┌─────────────────────────────────┐
│  ○ Daemon not running           │
│                                 │
│  $ alpaca start              📋 │  ← Copy button
├─────────────────────────────────┤
│  ⚙ Preferences...               │
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Gray status indicator
- Command to start daemon with copy button
- Access to preferences and quit

### State 2: Daemon Running, No Model Loaded (Idle)

```
┌─────────────────────────────────┐
│  ○ Idle                         │  ← Yellow indicator
│  No model loaded                │
├─────────────────────────────────┤
│  ▶ Load Model...                │  → Preset submenu
├─────────────────────────────────┤
│  ⚙ Preferences...               │
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Yellow status indicator
- "Load Model..." opens preset selection submenu

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
│  ⚙ Preferences...               │
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Green status indicator
- Current preset name and endpoint displayed
- "Open in Browser" opens the llama-server web UI
- "Switch Model..." for quick switching
- "Stop" to unload model

### State 4: Loading/Switching Model

```
┌─────────────────────────────────┐
│  ◐ Loading...                   │  ← Animated indicator
│  mistral-7b-q4                  │  ← Target preset name
├─────────────────────────────────┤
│  ✕ Cancel                       │
├─────────────────────────────────┤
│  ⚙ Preferences...               │
│  ⌘ Quit Alpaca                  │
└─────────────────────────────────┘
```

- Animated loading indicator
- Shows target preset name
- Cancel option to abort loading

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

## Preferences Window

Minimal settings window accessible from "Preferences...":

```
┌─────────────────────────────────────────────┐
│  Alpaca Preferences                    [x]  │
├─────────────────────────────────────────────┤
│                                             │
│  llama-server path:                         │
│  ┌─────────────────────────────────┐        │
│  │ /usr/local/bin/llama-server    │ [...]  │
│  └─────────────────────────────────┘        │
│                                             │
│  Default port:                              │
│  ┌─────────────────────────────────┐        │
│  │ 8080                            │        │
│  └─────────────────────────────────┘        │
│                                             │
│                            [Cancel] [Save]  │
└─────────────────────────────────────────────┘
```

Settings:
- Path to llama-server binary
- Default port for llama-server

Note: Preset management is done via CLI or direct YAML editing.

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

### Copy Command Button

In "Daemon not running" state:
- Click copies `alpaca start` to clipboard
- Brief visual feedback (button text changes to "Copied!")

### Quit Alpaca

- If model is running, stop it first
- Close GUI application
- Note: Does NOT stop the daemon (daemon runs independently)
