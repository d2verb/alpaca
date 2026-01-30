# Architecture

## Overview

Alpaca consists of three main components:

```
┌─────────────────────────────────────┐
│         alpaca (Go binary)          │
│                                     │
│  CLI Commands:                      │
│  - alpaca start [--foreground]      │
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

The daemon can run in two modes:
- **Background mode** (default): Detaches from terminal, writes logs to files
- **Foreground mode** (`--foreground` flag): Runs in current terminal, useful for debugging

### GUI (macOS Menu Bar App)

Native macOS app written in SwiftUI. Provides quick access to common operations.

Features:
- Show current status in menu bar
- Quick model switching
- Minimal preferences window

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
{
  "command": "load",
  "args": {
    "identifier": "p:codellama-7b"
  }
}
```

**Response Format:**
```json
{
  "status": "ok",
  "data": {
    "endpoint": "http://127.0.0.1:8080"
  }
}
```

**Error Response:**
```json
{
  "status": "error",
  "error": "model not found",
  "error_code": "model_not_found"
}
```

**Error Codes:**
Error responses include an `error_code` field for programmatic error handling:
- `preset_not_found` - Requested preset does not exist
- `model_not_found` - Model file not found or HuggingFace model not downloaded
- `server_failed` - llama-server failed to start or health check timed out

**Available Commands:**

**`status`** - Get current daemon state and loaded model info

Response (when running):
```json
{
  "status": "ok",
  "data": {
    "state": "running",
    "preset": "codellama-7b",
    "endpoint": "http://127.0.0.1:8080"
  }
}
```

Response (when idle):
```json
{
  "status": "ok",
  "data": {
    "state": "idle"
  }
}
```

**`load`** - Load a model (format: `h:org/repo:quant`, `p:preset-name`, or `f:/path/to/file`)

Request:
```json
{
  "command": "load",
  "args": {
    "identifier": "p:codellama-7b"
  }
}
```

Response:
```json
{
  "status": "ok",
  "data": {
    "endpoint": "http://127.0.0.1:8080"
  }
}
```

**`unload`** - Stop the currently running model

Response:
```json
{
  "status": "ok"
}
```

**`list_presets`** - List all available presets

Response:
```json
{
  "status": "ok",
  "data": {
    "presets": ["codellama-7b", "mistral-7b", "llama2-13b"]
  }
}
```

**`list_models`** - List all downloaded models

Response:
```json
{
  "status": "ok",
  "data": {
    "models": [
      {
        "repo": "TheBloke/CodeLlama-7B-GGUF",
        "quant": "Q4_K_M",
        "size": 4368438272
      }
    ]
  }
}
```

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
4. Fork background process with `--foreground` flag
5. Background process:
   - Writes PID file (`~/.alpaca/alpaca.pid`)
   - Sets up log rotation for `daemon.log` and `llama.log`
   - Creates Unix socket listener
   - Enters idle state (no model loaded)

**Foreground Mode:**
```bash
$ alpaca start --foreground
```
Runs daemon in current terminal without detaching. Useful for debugging.

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
Loads model using settings from `~/.alpaca/presets/my-preset.yaml`.

**2. Via HuggingFace Format:**
```bash
$ alpaca load h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```
Loads model using metadata from `~/.alpaca/models/.metadata.json`. If not downloaded, auto-pulls first.

**3. Via File Path:**
```bash
$ alpaca load f:~/models/my-model.gguf
```
Loads model file directly with default settings from `~/.alpaca/config.yaml`.

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

```
idle → loading → running
  ↑                ↓
  └────────────────┘
      (unload)
```

- **idle**: No model loaded
- **loading**: Model is starting (llama-server not ready)
- **running**: Model is ready and serving

**Concurrency and Lock-Free Reads:**

State and preset information are managed using atomic operations (`atomic.Value` and `atomic.Pointer`), enabling lock-free concurrent reads. This design ensures that:
- CLI `status` commands return immediately without blocking
- GUI can poll status frequently without timeout issues
- State queries never wait for model loading operations

The `Run()` method acquires a mutex to serialize model loading operations, but state reads via `State()` and `CurrentPreset()` methods remain lock-free and return instantly, even during long model loading operations (e.g., large models taking >30 seconds to initialize).

## File System Layout

```
~/.alpaca/
├── config.yaml              # User configuration
├── alpaca.sock              # Unix socket for IPC
├── alpaca.pid               # Daemon PID file
├── presets/                 # Preset YAML files
│   ├── codellama-7b.yaml
│   └── llama2-13b.yaml
├── models/                  # Downloaded GGUF files
│   ├── .metadata.json       # Model metadata database
│   ├── codellama-7b-instruct.Q4_K_M.gguf
│   └── llama-2-13b-chat.Q5_K_M.gguf
└── logs/                    # Log files
    ├── daemon.log           # Daemon log (with rotation)
    └── llama.log            # llama-server output (with rotation)
```

### Logging System

Alpaca uses structured logging with automatic rotation:

- **daemon.log**: Daemon lifecycle events (startup, shutdown, errors)
  - Format: Structured text logs (slog)
  - Rotation: 50MB max size, 3 backups, 7 days retention

- **llama.log**: llama-server stdout/stderr
  - Format: Raw llama-server output
  - Rotation: Same as daemon.log

Both logs use `lumberjack` for rotation and compression.

### Model Metadata System

Models downloaded via `alpaca pull` are tracked in `~/.alpaca/models/.metadata.json`:

```json
{
  "models": [
    {
      "repo": "TheBloke/CodeLlama-7B-GGUF",
      "quant": "Q4_K_M",
      "filename": "codellama-7b.Q4_K_M.gguf",
      "size": 4368438272,
      "downloaded_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

This metadata enables:
- Loading models via `repo:quant` identifier
- Listing downloaded models with `alpaca ls`
- Tracking download history

## Cross-Platform Considerations

- **CLI and Daemon**: Written in Go, naturally cross-platform
- **GUI**: OS-specific implementation
  - macOS: SwiftUI (current)
  - Windows: Future (WinUI or similar)
  - Linux: Future (GTK or similar)

The Unix socket approach works on macOS and Linux. For Windows, named pipes would be the equivalent.
