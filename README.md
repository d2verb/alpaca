# Alpaca

A lightweight wrapper around [llama-server](https://github.com/ggerganov/llama.cpp) for macOS.

## Why Alpaca?

- **Preset system**: Save model + argument combinations as presets
- **Easy model switching**: Switch models without manually restarting servers
- **Full llama-server options**: Access all llama-server arguments via `extra_args`
- **CLI + GUI**: Command-line interface and macOS menu bar app

## Installation

```bash
# Install via Homebrew (coming soon)
brew install d2verb/tap/alpaca

# Or build from source
go build -o alpaca ./cmd/alpaca
```

## Quick Start

```bash
# Start the daemon
alpaca start

# Download a model
alpaca model pull h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q4_K_M

# Create a preset (~/.alpaca/presets/mistral.yaml)
cat <<EOF > ~/.alpaca/presets/mistral.yaml
model: f:~/.alpaca/models/mistral-7b-instruct-v0.2.Q4_K_M.gguf
context_size: 4096
gpu_layers: 35
EOF

# Load the model
alpaca load p:mistral

# Check status
alpaca status

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

## Requirements

- macOS
- [llama-server](https://github.com/ggerganov/llama.cpp) installed and available in PATH

## License

MIT
