<div align="center">
  <img src="alpaca-icon.png" alt="Alpaca Logo" width="200"/>
  <h1>Alpaca</h1>
  <p>A lightweight wrapper around <a href="https://github.com/ggerganov/llama.cpp">llama-server</a> for macOS</p>

  [![CI](https://github.com/d2verb/alpaca/actions/workflows/ci.yml/badge.svg)](https://github.com/d2verb/alpaca/actions/workflows/ci.yml)
  [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
</div>

## Why Alpaca?

- **Preset system**: Save model + argument combinations as presets
- **Interactive preset creation**: Create presets with guided prompts (`alpaca preset new`)
- **Easy model switching**: Switch models without manually restarting servers
- **Full llama-server options**: Access all llama-server arguments via `extra_args`
- **Model management**: Download, list, and remove HuggingFace models
- **Log viewing**: View daemon and server logs with follow mode (`alpaca logs -f`)
- **CLI + GUI**: Command-line interface and macOS menu bar app
- **Automatic log rotation**: Logs rotate at 50MB with compression

## Installation

```bash
# Build from source
# Requires Go 1.23+ and Task (https://taskfile.dev)
task build
# Binary will be at ./build/alpaca
```

## Quick Start

```bash
# Start the daemon
alpaca start

# Download a model
alpaca model pull h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q4_K_M

# Create a preset interactively
alpaca preset new
# Or create manually (~/.alpaca/presets/mistral.yaml)
cat <<EOF > ~/.alpaca/presets/mistral.yaml
model: f:~/.alpaca/models/mistral-7b-instruct-v0.2.Q4_K_M.gguf
context_size: 4096
gpu_layers: 35
EOF

# View preset details
alpaca preset show mistral

# Load the model
alpaca load p:mistral

# Check status
alpaca status

# View logs (follow mode)
alpaca logs -f

# Stop the model
alpaca unload

# Stop the daemon
alpaca stop
```

## Preset Format

```yaml
# File path
model: f:~/.alpaca/models/your-model.gguf
context_size: 4096
gpu_layers: 35
threads: 8
port: 8080
extra_args:
  - "--flash-attn"
  - "--cont-batching"

# Or HuggingFace format (auto-resolved)
model: h:TheBloke/Mistral-7B-GGUF:Q4_K_M
context_size: 4096
gpu_layers: 35
```

## Commands

### Daemon Management
- `alpaca start [--foreground]` - Start the daemon
- `alpaca stop` - Stop the daemon
- `alpaca status` - Show current status
- `alpaca logs [-f] [-d|-s]` - View daemon or server logs

### Model Management
- `alpaca load <identifier>` - Load a model (`h:`, `p:`, or `f:` prefix)
- `alpaca unload` - Stop the current model
- `alpaca model pull h:org/repo:quant` - Download a model
- `alpaca model ls` - List downloaded models
- `alpaca model rm h:org/repo:quant` - Remove a model

### Preset Management
- `alpaca preset ls` - List presets
- `alpaca preset show <name>` - Show preset details
- `alpaca preset new` - Create preset interactively
- `alpaca preset rm <name>` - Remove a preset

### Other
- `alpaca version` - Show version information

For detailed command documentation, see [`docs/design/cli.md`](docs/design/cli.md).

## Requirements

- **macOS** (primary platform)
- **Go 1.23+** (for building from source)
- **[llama-server](https://github.com/ggerganov/llama.cpp)** installed and available in PATH

## Documentation

- [Architecture Overview](docs/design/architecture.md)
- [CLI Command Reference](docs/design/cli.md)
- [Preset Format](docs/design/preset-format.md)
- [GUI Documentation](docs/design/gui.md)

## License

MIT
