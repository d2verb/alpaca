# Development Guide

## Tech Stack

| Component | Technology |
|-----------|------------|
| CLI / Daemon | Go 1.25+ |
| CLI Framework | [kong](https://github.com/alecthomas/kong) |
| GUI | SwiftUI (Swift 6.0+, Xcode 16+) |
| Task Runner | [Task](https://taskfile.dev/) |
| Git Hooks | [lefthook](https://github.com/evilmartians/lefthook) |
| CI/CD | GitHub Actions |
| Release (Go) | GoReleaser |

## Project Structure

```
alpaca/
├── cmd/                        # CLI entry point
│   └── alpaca/
│       └── main.go
├── internal/                   # Private packages
│   ├── daemon/                 # Daemon logic
│   │   ├── daemon.go
│   │   ├── server.go           # Unix socket server
│   │   └── handler.go          # Command handlers
│   ├── client/                 # Daemon client (for CLI)
│   │   └── client.go
│   ├── preset/                 # Preset management
│   │   ├── preset.go
│   │   └── loader.go
│   ├── llama/                  # llama-server management
│   │   ├── process.go
│   │   └── health.go
│   ├── pull/                   # HuggingFace download
│   │   └── pull.go
│   ├── config/                 # Configuration
│   │   └── config.go
│   └── protocol/               # Daemon communication protocol
│       └── protocol.go
├── gui/                        # SwiftUI app
│   └── Alpaca/
│       ├── Alpaca.xcodeproj
│       ├── Sources/
│       │   ├── AlpacaApp.swift
│       │   ├── MenuBarView.swift
│       │   ├── DaemonClient.swift
│       │   └── ...
│       └── Resources/
├── docs/
│   └── design/
├── scripts/                    # Utility scripts
├── .github/
│   └── workflows/
│       ├── ci.yml
│       └── release.yml
├── go.mod
├── go.sum
├── Taskfile.yml
├── .golangci.yml
└── .lefthook.yml
```

## Coding Rules

### Go

| Item | Rule |
|------|------|
| Formatter | `goimports` (enforced in CI) |
| Linter | `golangci-lint` with custom config |
| Error handling | Wrap with `fmt.Errorf("context: %w", err)` |
| Naming | Go standard (MixedCaps, short variable names) |
| Package names | Singular, short (`preset` not `presets`) |
| Comments | GoDoc format for public APIs |

### Swift

| Item | Rule |
|------|------|
| Formatter | `swift-format` |
| Linter | SwiftLint (default rules) |
| Naming | Swift API Design Guidelines |
| Concurrency | async/await + Actor (Swift 6) |

## Commit Convention

Use [Gitmoji](https://gitmoji.dev/) for commit messages.

### Format

```
<emoji> <subject>

<body (optional)>
```

### Common Emojis

| Emoji | Code | Usage |
|-------|------|-------|
| ✨ | `:sparkles:` | New feature |
| 🐛 | `:bug:` | Bug fix |
| ♻️ | `:recycle:` | Refactor |
| 📝 | `:memo:` | Documentation |
| ✅ | `:white_check_mark:` | Add/update tests |
| 🔧 | `:wrench:` | Configuration |
| 🎨 | `:art:` | Code style/format |
| 🚀 | `:rocket:` | Performance |
| 🔥 | `:fire:` | Remove code/files |
| 🏗️ | `:building_construction:` | Architecture changes |

### Examples

```
✨ Add alpaca run command
🐛 Fix preset loading when path contains spaces
♻️ Extract llama-server process management to separate package
📝 Document CLI commands
```

## Branch Strategy

**GitHub Flow**

```
main (always deployable)
  │
  ├── feature/add-pull-command
  │     ↓ PR & merge
  ├── fix/preset-loading-bug
  │     ↓ PR & merge
  └── ...
```

### Rules

1. `main` is always deployable
2. Create feature branch from `main`
3. Open PR for review
4. Merge to `main` after CI passes
5. Delete feature branch after merge

### Branch Naming

```
feature/<description>   # New feature
fix/<description>       # Bug fix
docs/<description>      # Documentation
refactor/<description>  # Refactoring
```

## Development Workflow

### Setup

```bash
# Install Go dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest

# Install Task
brew install go-task

# Install lefthook and set up hooks
brew install lefthook
lefthook install
```

### Daily Development

```bash
# Run all checks
task check

# Run tests
task test

# Build CLI
task build

# Run linter
task lint

# Format code
task fmt
```

### Before Commit

lefthook automatically runs:
- `goimports` (format)
- `golangci-lint` (lint)
- Gitmoji commit message validation

## CI Pipeline

### On Pull Request

```yaml
jobs:
  lint:
    - golangci-lint
    - swift-format --lint
    - swiftlint

  test:
    - go test -race ./...

  build:
    - go build ./cmd/alpaca
    - xcodebuild (GUI)
```

### On Push to Main

```yaml
jobs:
  # Same as PR
  lint: ...
  test: ...
  build: ...
```

### On Tag (Release)

```yaml
jobs:
  release:
    - GoReleaser (CLI binary)
    - Xcode archive (GUI app)
```

## Daemon Communication Protocol

CLI and GUI communicate with the daemon via Unix socket using JSON.

### Request Format

```json
{
  "command": "<command_name>",
  "args": { ... }
}
```

### Response Format

Success:
```json
{
  "status": "ok",
  "data": { ... }
}
```

Error:
```json
{
  "status": "error",
  "error": "<error_message>"
}
```

### Commands

| Command | Args | Response Data |
|---------|------|---------------|
| `status` | - | `{"state": "running", "preset": "...", "endpoint": "..."}` |
| `run` | `{"preset": "name"}` | `{"endpoint": "http://localhost:8080"}` |
| `kill` | - | `{}` |
| `list_presets` | - | `{"presets": ["name1", "name2"]}` |

### Example

```json
// Request
{"command": "run", "args": {"preset": "codellama-7b"}}

// Response (success)
{"status": "ok", "data": {"endpoint": "http://localhost:8080"}}

// Response (error)
{"status": "error", "error": "preset 'codellama-7b' not found"}
```
