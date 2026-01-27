# CLAUDE.md

Guidance for Claude Code when working in this repository.

## Project Overview

Alpaca is a lightweight wrapper around `llama-server` (from llama.cpp) for macOS. It provides CLI and GUI interfaces, similar to Ollama, but with full access to llama-server's options and better performance.

**Key Goals:**
- Thin wrapper (proxy tool approach, not a new inference engine)
- Preset system for model + argument combinations
- Smooth model switching without manual server restarts
- Full llama-server option support

## Tech Stack

- CLI / Daemon: Go 1.23+, kong (CLI framework)
- GUI: SwiftUI (Swift 6.0+, macOS menu bar app)
- Communication: Unix socket with JSON protocol
- Task Runner: Task (Taskfile.yml)

## Common Commands

```bash
task build      # Build CLI binary
task test       # Run tests with coverage
task lint       # Run golangci-lint
task check      # Run fmt + lint + test
task gui:open   # Open Xcode project
```

## Design Documents

Detailed specifications are in `docs/design/`. **Read before implementing:**

- `architecture.md` - Component architecture, daemon lifecycle
- `cli.md` - CLI command reference
- `gui.md` - GUI layouts and states
- `preset-format.md` - Preset YAML schema
- `mvp.md` - MVP scope definition

## Daemon Protocol

CLI/GUI communicate with daemon via Unix socket (`~/.alpaca/alpaca.sock`).

```json
// Request
{"command": "run", "args": {"preset": "codellama-7b"}}

// Response
{"status": "ok", "data": {"endpoint": "http://localhost:8080"}}
```

Commands: `status`, `run`, `kill`, `list_presets`

## Key Behaviors

**Model Switching:**
1. Send SIGTERM to current llama-server
2. Wait for graceful shutdown (max 10s, then SIGKILL)
3. Start llama-server with new preset
4. Wait for /health to report ready

**Preset Loading:**
- Presets are YAML files in `~/.alpaca/presets/`
- `extra_args` field passes arbitrary flags to llama-server

## Scope Boundaries

**Do:**
- Keep it simple (thin wrapper)
- Use system-installed llama-server
- Rely on llama-server's /health for health checks
- Write tests before implementing new features (TDD approach)

**Do Not:**
- Add llama.cpp version management
- Implement custom inference
- Over-engineer
