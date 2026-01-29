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
| `model` | string | Model identifier with explicit prefix: `h:org/repo:quant` (HuggingFace) or `f:/path/to/file` (file path). |

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

### Basic Preset (File Path)

```yaml
# Absolute path
model: "f:/Users/username/.alpaca/models/mistral-7b.Q4_K_M.gguf"
context_size: 4096
gpu_layers: 35
```

```yaml
# Home directory
model: "f:~/.alpaca/models/mistral-7b.Q4_K_M.gguf"
context_size: 4096
gpu_layers: 35
```

```yaml
# Relative to preset file
model: "f:./models/codellama.gguf"
context_size: 4096
gpu_layers: 35
```

### Preset with HuggingFace Model Reference

```yaml
# HuggingFace format (auto-resolved at runtime)
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
context_size: 4096
gpu_layers: 35
```

**Note:** HuggingFace models must be downloaded first with `alpaca model pull h:org/repo:quant`. The model field will be automatically resolved to `f:/path/to/downloaded/file.gguf` at runtime.

### Full-Featured Preset

```yaml
model: "f:~/.alpaca/models/codellama-34b-instruct.Q4_K_M.gguf"
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
model: "f:~/.alpaca/models/llama3-8b.Q4_K_M.gguf"
host: "0.0.0.0"  # Listen on all interfaces
port: 8080
context_size: 4096
gpu_layers: 35
```

## Model Field Resolution

The `model` field requires an explicit prefix to indicate the identifier type:

### 1. File Paths (`f:`)

File paths must use the `f:` prefix and support absolute, home, and relative paths:

```yaml
model: "f:/abs/path/model.gguf"        # Absolute path
model: "f:~/models/model.gguf"         # Home directory expansion
model: "f:./model.gguf"                # Relative to preset file directory
model: "f:../shared/model.gguf"        # Parent directory
```

The `f:` prefix is stripped when passing the path to llama-server.

### 2. HuggingFace Format (`h:`)

HuggingFace models must use the `h:` prefix:

```yaml
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M"
```

**Resolution process:**
1. Model must be downloaded first with `alpaca model pull h:org/repo:quant`
2. At runtime, `h:org/repo:quant` is resolved to `f:/path/to/downloaded/file.gguf`
3. The `f:` prefix is stripped when starting llama-server

**Error handling:**
- Missing prefix → Parse error with clear message
- HuggingFace model not downloaded → Error with suggestion to run `alpaca model pull`
- File path doesn't exist → Error when starting llama-server

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
