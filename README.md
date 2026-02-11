<div align="center">
  <img src="alpaca-icon.png" alt="Alpaca Logo" width="200"/>
  <h1>Alpaca</h1>
  <p>A lightweight wrapper around <a href="https://github.com/ggerganov/llama.cpp">llama-server</a></p>

  [![CI](https://github.com/d2verb/alpaca/actions/workflows/ci.yml/badge.svg)](https://github.com/d2verb/alpaca/actions/workflows/ci.yml)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
</div>

## Why Alpaca?

- **Preset system**: Save model + argument combinations as reusable presets
- **Router mode**: Run multiple models simultaneously (chat + embedding, A/B testing)
- **Easy model switching**: Switch models without manually restarting servers
- **Vision/Audio support**: Automatically detects and configures multimodal models
- **Full llama-server options**: Pass any llama-server argument via the `options` map
- **HuggingFace integration**: Download models directly with `alpaca pull`

## Demo

![Alpaca Demo](demo.gif)

## Requirements

- **llama-server** installed and available in PATH
- **macOS or Linux** (GUI is macOS only)

## Installation

### Homebrew (macOS)

```bash
brew install d2verb/tap/alpaca
```

### Shell script (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/d2verb/alpaca/main/install.sh | sh
```

### Build from source

```bash
# Requires Go 1.25+ and Task (https://taskfile.dev)
task build
# Binary will be at ./build/alpaca
```

## Quick Start

```bash
# Start the daemon
alpaca start

# Download and load a model
alpaca pull h:unsloth/gemma-3-4b-it-GGUF:Q4_K_M
alpaca load h:unsloth/gemma-3-4b-it-GGUF:Q4_K_M

# Check status
alpaca status
# → Running at http://localhost:8080

# Create a preset for repeated use
alpaca new

# Enable shell completion (zsh, fish also supported)
echo "$(alpaca completion-script)" >> ~/.bashrc
```

## Commands

### Daemon

- `alpaca start` - Start the daemon
- `alpaca stop` - Stop the daemon
- `alpaca status` - Show current status
- `alpaca open` - Open llama-server in browser
- `alpaca logs [-f] [-s]` - View logs (`-f` follow, `-s` server logs)

### Models

- `alpaca load <identifier>` - Load a model (`p:preset`, `h:org/repo:quant`, `f:path`)
- `alpaca unload` - Stop the current model
- `alpaca pull h:org/repo:quant` - Download a model
- `alpaca ls` - List presets and models
- `alpaca show <identifier>` - Show preset or model details
- `alpaca rm <identifier>` - Remove a preset or model
- `alpaca new` - Create a preset interactively (single or router mode)
- `alpaca edit [identifier]` - Open a preset in your editor

### Utility

- `alpaca upgrade [-c]` - Upgrade to the latest version (`-c` check only)
- `alpaca version` - Show version
- `alpaca completion-script` - Output shell completion script

## Documentation

- [CLI Reference](docs/design/cli.md)
- [Preset Format](docs/design/preset-format.md)
- [Architecture](docs/design/architecture.md)

## License

MIT
