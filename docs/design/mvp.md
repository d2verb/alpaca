# MVP Scope

## Overview

This document defines the Minimum Viable Product (MVP) scope for Alpaca.

## MVP Features

### CLI

| Command | Description |
|---------|-------------|
| `alpaca start` | Start the daemon |
| `alpaca stop` | Stop the daemon |
| `alpaca status` | Show current status |
| `alpaca load p:<preset>` | Load a model with the specified preset |
| `alpaca unload` | Stop the currently running model |
| `alpaca preset list` | List available presets |
| `alpaca model pull h:<repo>:<quant>` | Download model from HuggingFace |

### Daemon

- Unix socket communication (`~/.alpaca/alpaca.sock`)
- llama-server process management (start/stop)
- Model switching with graceful shutdown
- State management (idle / loading / running)

### GUI (Minimal)

- Menu bar icon
- Status display (running / idle / daemon not running)
- Preset selection for model switching
- Stop button
- "Daemon not running" guidance

### Other

- Preset YAML loading (manually created)
- `~/.alpaca/` directory structure initialization

## Out of Scope (Phase 2+)

### CLI
- `alpaca preset create` - Interactive preset creation
- `alpaca preset edit` - Edit existing preset
- `alpaca preset delete` - Delete preset
- `alpaca model list` - List downloaded models

### GUI
- Preferences window (llama-server path, default port settings)

## Implementation Phases

### Phase 1: MVP
Core functionality to run llama-server with presets.

### Phase 2: Convenience Features
- Preset management commands
- Model management commands
- GUI preferences window

### Phase 3: Polish
- Error handling improvements
- Better UX for edge cases
- Documentation and examples
