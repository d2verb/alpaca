# Architecture

## Overview

Alpaca consists of three main components:

```text
┌─────────────────────────────────────┐
│         alpaca (Go binary)          │
│                                     │
│  CLI Commands:                      │
│  - alpaca start                     │
│  - alpaca stop                      │
│  - alpaca status                    │
│  - alpaca load <model>              │
│  - alpaca unload                    │
│                                     │
│  When started:                      │
│  ┌─────────────────────────────┐   │
│  │  Daemon (background mode)   │   │
│  │  - Listens on Unix socket   │   │
│  │  - Manages llama-server     │   │
│  │  - Handles IPC requests     │   │
│  └──────────────┬──────────────┘   │
└─────────────────┼───────────────────┘
                  │ Unix socket
                  │ (~/.alpaca/alpaca.sock)
                  │
┌─────────────────┼───────────────────┐
│                 ▼                   │
│          GUI (SwiftUI)              │
│       macOS Menu Bar App            │
└─────────────────────────────────────┘
                  │
                  │ Process management
                  ▼
          ┌───────────────┐
          │ llama-server  │
          └───────────────┘
```

## Components

### CLI (alpaca)

Command-line interface written in Go. Communicates with the daemon via Unix socket.

Primary interface for:
- Starting/stopping the daemon (`alpaca start`, `alpaca stop`)
- Managing presets (show, new)
- Listing resources (`alpaca ls`)
- Downloading models (`alpaca pull`)
- Removing resources (`alpaca rm`)
- Loading/unloading models (`alpaca load`, `alpaca unload`)
- Viewing status and logs (`alpaca status`, `alpaca logs`)
- Version information (`alpaca version`)

### Daemon

Background process written in Go. Started via `alpaca start`, runs as a daemon by default.

Responsibilities:
- Maintain llama-server process
- Handle model switching (graceful shutdown → restart)
- Serve status information to CLI and GUI
- Listen on Unix socket for commands
- Manage logging (daemon.log, llama.log)

The daemon runs in background mode by default, detaching from the terminal and writing logs to files.

### GUI (macOS Menu Bar App)

Native macOS app written in SwiftUI. Provides quick access to common operations.

Features:
- Show current status in menu bar
- Quick model switching
- Minimal preferences window

## Router Mode (Multi-Model)

Alpaca supports running multiple models simultaneously via llama-server's router mode (requires llama-server b7350+).

### Architecture

In router mode, a single llama-server process manages multiple models as child processes:

```text
alpaca daemon
  └── llama-server (router process)
        ├── Child: Model A (qwen3)          ← independent KV-cache
        ├── Child: Model B (nomic-embed)    ← independent KV-cache
        └── Child: Model C (gemma3)         ← independent KV-cache
```

Key properties:
- **Crash isolation**: One model crashing doesn't affect others
- **LRU eviction**: When `--models-max` is reached, the least recently used model is auto-unloaded
- **Single endpoint**: All models served from one port with model selection via API

### Config Generation

Alpaca generates a `config.ini` file from the router preset YAML and passes it via `--models-preset`:

```text
Preset YAML → GenerateConfigINI() → ~/.alpaca/router-config.ini → llama-server --models-preset
```

- The config file is atomically written (temp file + rename) on each `daemon.Run()`
- Cleaned up on `daemon.Kill()` (best-effort)
- HuggingFace model references (`h:`) are resolved to file paths before config generation

### Model Status

In router mode, the daemon queries llama-server's `/models` API to get per-model status (loaded/loading/unloaded). This information is included in the status IPC response for both CLI and GUI.

```text
CLI → [IPC: status] → Daemon → [HTTP: GET /models] → llama-server
                             ← merged response with model statuses
```

## Communication

### Unix Socket

All communication between CLI/GUI and Daemon uses Unix socket at `~/.alpaca/alpaca.sock`.

Reasons for choosing Unix socket:
- Slightly faster than HTTP over TCP (~0.1ms vs ~0.5-1ms)
- Secure by default (file permissions)
- No network exposure risk

Note: The actual bottleneck is llama-server inference time (hundreds of ms to seconds), so communication overhead is negligible.

### Protocol

JSON-based request/response protocol over Unix socket. Messages are newline-delimited.

**Request Format:**
```json
{"command": "<command>", "args": {...}}
```

**Response Format:**
```json
{"status": "ok", "data": {...}}
{"status": "error", "error": "<message>", "error_code": "<code>"}
```

**Available Commands:**
- `status` - Get daemon state and loaded model info
- `load` - Load a model (`h:org/repo:quant`, `p:preset-name`, or `f:/path`)
- `unload` - Stop the currently running model
- `list_presets` - List available presets
- `list_models` - List downloaded models

**Error Codes:**
- `preset_not_found` - Requested preset does not exist
- `model_not_found` - Model file not found
- `server_failed` - llama-server failed to start

## Daemon Lifecycle

### Starting the Daemon

```bash
$ alpaca start
# Daemon started (PID: 12345)
# Logs: ~/.alpaca/logs/daemon.log
```

Process:
1. Check if daemon is already running (via PID file)
2. Clean up stale socket/PID files if found
3. Create required directories (`~/.alpaca`, `~/.alpaca/logs`, etc.)
4. Fork background process with internal `--daemon` flag
5. Background process:
   - Writes PID file (`~/.alpaca/alpaca.pid`)
   - Sets up log rotation for `daemon.log` and `llama.log`
   - Creates Unix socket listener
   - Enters idle state (no model loaded)

There is no foreground mode. The daemon always runs in the background.

### Stopping the Daemon

```bash
$ alpaca stop
```

Process:
1. Read PID from `~/.alpaca/alpaca.pid`
2. Send SIGTERM to daemon process
3. Wait for graceful shutdown (max 10 seconds)
4. If timeout, send SIGKILL
5. Remove PID file and socket

The daemon gracefully stops any running llama-server before exiting.

### GUI without Daemon

If GUI is launched without daemon running:
- Show "Daemon not running" message
- Prompt user to run `alpaca start`

## Model Loading and Switching

### Loading a Model

Models can be loaded in two ways:

**1. Via Preset:**
```bash
$ alpaca load p:my-preset
```
Loads model using settings from the global preset named `my-preset`.

**2. Via HuggingFace Format:**
```bash
$ alpaca load h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```
Loads model using metadata from `~/.alpaca/models/.metadata.json`. If not downloaded, auto-pulls first.

**3. Via File Path:**
```bash
$ alpaca load f:~/models/my-model.gguf
```
Loads model file directly with default settings (host: 127.0.0.1, port: 8080, context_size: 4096).

### Model Switching Flow

When switching models (loading while another is running):

1. Acquire daemon lock
2. Stop current llama-server if running:
   - Send SIGTERM to llama-server process
   - Wait for graceful shutdown (max 10 seconds)
   - Force kill if timeout
3. Load preset or create preset from HF format
4. Start new llama-server process with preset args
5. Pipe llama-server output to `~/.alpaca/logs/llama.log`
6. Wait for `/health` endpoint to report ready
7. Update daemon state to `running`
8. Release lock

### State Transitions

```text
idle → loading → running
  ↑                ↓
  └────────────────┘
      (unload)
```

- **idle**: No model loaded
- **loading**: Model is starting (llama-server not ready)
- **running**: Model is ready and serving

## Cross-Platform Considerations

- **CLI and Daemon**: Written in Go, naturally cross-platform
- **GUI**: OS-specific implementation
  - macOS: SwiftUI (current)
  - Windows: Future (WinUI or similar)
  - Linux: Future (GTK or similar)

The Unix socket approach works on macOS and Linux. For Windows, named pipes would be the equivalent.
