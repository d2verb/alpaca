# Preset Format

## Overview

Presets define a model + argument combination that can be loaded with a single command. They are stored as YAML files in `~/.alpaca/presets/`.

## File Location

```
~/.alpaca/presets/
├── codellama-7b-q4.yaml
├── mistral-7b-q4.yaml
└── deepseek-coder.yaml
```

Preset name is derived from the filename (without `.yaml` extension).

## Format

```yaml
# Required: path to the model file
model: ~/.alpaca/models/codellama-7b-Q4_K_M.gguf

# Common options (mapped to llama-server arguments)
context_size: 4096      # --ctx-size
gpu_layers: 35          # --n-gpu-layers
threads: 8              # --threads
port: 8080              # --port

# Additional llama-server arguments
# Use this for any option not explicitly defined above
extra_args:
  - "--flash-attn"
  - "--cont-batching"
  - "--mlock"
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `model` | string | Path to GGUF model file. Supports `~` expansion. |

### Optional Fields (Common)

| Field | Type | Default | llama-server flag |
|-------|------|---------|-------------------|
| `context_size` | int | 2048 | `--ctx-size` |
| `gpu_layers` | int | 0 | `--n-gpu-layers` |
| `threads` | int | (auto) | `--threads` |
| `port` | int | 8080 | `--port` |
| `host` | string | "127.0.0.1" | `--host` |

### Extra Arguments

The `extra_args` field accepts a list of strings that are passed directly to llama-server. This allows using any llama-server option, including new options from future llama.cpp releases.

```yaml
extra_args:
  - "--flash-attn"
  - "--cont-batching"
  - "--rope-scaling"
  - "linear"
  - "--rope-freq-base"
  - "10000"
```

## Examples

### Basic Preset

```yaml
model: ~/.alpaca/models/mistral-7b-instruct-v0.2.Q4_K_M.gguf
context_size: 4096
gpu_layers: 35
```

### Full-Featured Preset

```yaml
model: ~/.alpaca/models/codellama-34b-instruct.Q4_K_M.gguf
context_size: 8192
gpu_layers: 50
threads: 12
port: 8081
extra_args:
  - "--flash-attn"
  - "--cont-batching"
  - "--mlock"
  - "--no-mmap"
```

### Preset with Custom Host

```yaml
model: ~/.alpaca/models/llama3-8b.Q4_K_M.gguf
host: "0.0.0.0"  # Listen on all interfaces
port: 8080
context_size: 4096
gpu_layers: 35
```

## Design Decisions

### Why YAML?

- Human-readable and editable
- Supports comments
- Widely understood format
- Good library support in Go

### Why not Ollama's Modelfile format?

- YAML is more standard and flexible
- Easier to parse and validate
- Better editor support (syntax highlighting, etc.)
- `extra_args` provides full llama-server compatibility

### Extensibility

The `extra_args` field ensures forward compatibility. When llama.cpp adds new options, users can immediately use them without waiting for Alpaca updates.
