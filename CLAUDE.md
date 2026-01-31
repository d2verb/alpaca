# CLAUDE.md

## Project Overview

Alpaca: lightweight wrapper around `llama-server` (llama.cpp) for macOS with CLI/GUI interfaces.

**Core Principle:** Thin proxy wrapper, not a new inference engine.

## Tech Stack

- CLI/Daemon: Go, kong
- GUI: SwiftUI (macOS menu bar app)
- Communication: Unix socket (`~/.alpaca/alpaca.sock`) + JSON protocol
- Task Runner: Taskfile.yml

## Commands

```bash
task build      # Build CLI
task test       # Go tests + coverage
task gui:test   # Swift tests + coverage
task lint       # golangci-lint + deadcode
task check      # fmt + lint + test
```

## Design Documents

Detailed specs in `docs/design/`. **Read before implementing.**

Temporary design/implementation memos go in `docs/wip/`.

## After Every Change

1. Run `task check` (Go) or `task gui:test` (Swift)
2. Update related docs (`docs/design/`, `README.md`)

## Scope Boundaries

**Do:** TDD, keep it simple, use system-installed llama-server

**Don't:** llama.cpp version management, custom inference, over-engineering, YAGNI violations
