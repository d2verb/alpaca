# Architecture

## Overview

Alpaca consists of three main components:

```
┌─────────────────────────────────────┐
│            alpaca (Go)              │
│  ┌─────────┐  ┌─────────────────┐   │
│  │   CLI   │  │     Daemon      │   │
│  │         │  │    (alpacad)    │   │
│  └────┬────┘  └────────┬────────┘   │
│       │                │            │
│       └───────┬────────┘            │
│               │                     │
└───────────────┼─────────────────────┘
                │ Unix socket
                │ (~/.alpaca/alpaca.sock)
                │
┌───────────────┼─────────────────────┐
│               ▼                     │
│        GUI (SwiftUI)                │
│     macOS Menu Bar App              │
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
- Starting/stopping the daemon
- Managing presets (create, edit, list, delete)
- Downloading models (`alpaca pull`)
- Loading/unloading models

### Daemon (alpacad)

Background process written in Go. Manages llama-server lifecycle.

Responsibilities:
- Maintain llama-server process
- Handle model switching (graceful shutdown → restart)
- Serve status information to CLI and GUI
- Listen on Unix socket for commands

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

TBD: JSON-based request/response protocol over Unix socket.

## Daemon Lifecycle

### Starting the Daemon

```
$ alpaca start
```

- Daemon starts and listens on Unix socket
- No model is loaded initially (idle state)

### Stopping the Daemon

```
$ alpaca stop
```

- Sends SIGTERM to running llama-server (if any)
- Waits for graceful shutdown
- Daemon process exits

### GUI without Daemon

If GUI is launched without daemon running:
- Show "Daemon not running" message
- Prompt user to run `alpaca start`

## Model Switching

When user requests a model switch:

```
1. Send SIGTERM to current llama-server
2. Wait for graceful shutdown (max 10 seconds)
3. If timeout, send SIGKILL
4. Start llama-server with new preset
5. Wait for /health endpoint to report ready
6. Notify CLI/GUI of success
```

## Cross-Platform Considerations

- **CLI and Daemon**: Written in Go, naturally cross-platform
- **GUI**: OS-specific implementation
  - macOS: SwiftUI (current)
  - Windows: Future (WinUI or similar)
  - Linux: Future (GTK or similar)

The Unix socket approach works on macOS and Linux. For Windows, named pipes would be the equivalent.
