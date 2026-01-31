# Preset Format

## Overview

Presets define a model + argument combination that can be loaded with a single command. They are stored as YAML files in `~/.alpaca/presets/`.

## File Location

```
~/.alpaca/presets/
├── a1b2c3d4e5f67890.yaml
├── 1234567890abcdef.yaml
└── fedcba0987654321.yaml
```

Preset files are stored with random filenames (16 hex characters). The `name` field inside the YAML file is used as the identifier for loading (e.g., `alpaca load p:codellama-7b`).

## Format

```yaml
# Required: preset identifier (alphanumeric, underscore, hyphen only)
name: codellama-7b

# Required: model identifier with explicit prefix
model: "f:~/.alpaca/models/codellama-7b-Q4_K_M.gguf"

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
| `name` | string | Preset identifier used for loading (e.g., `alpaca load p:name`). Must match `[a-zA-Z0-9_-]+`. |
| `model` | string | Model identifier with explicit prefix: `h:org/repo:quant` (HuggingFace) or `f:/path/to/file` (file path). |

### Optional Fields (Common)

| Field | Type | Default | llama-server flag |
|-------|------|---------|-------------------|
| `context_size` | int | 2048 | `--ctx-size` |
| `gpu_layers` | int | -1 (all layers) | `--n-gpu-layers` |
| `threads` | int | 0 (omit flag, llama-server decides) | `--threads` |
| `port` | int | 8080 | `--port` |
| `host` | string | "127.0.0.1" | `--host` |

**Note on defaults:** Alpaca applies explicit defaults for `context_size`, `gpu_layers`, `host`, and `port`. When these fields are omitted from the YAML, the defaults are still passed to llama-server. Only `threads` omits the flag when not specified, allowing llama-server to decide.

**Special case for `gpu_layers`:** Setting `gpu_layers: 0` in YAML will use 0 GPU layers (CPU-only mode). However, due to YAML's zero-value behavior, omitting the field also results in 0, which gets replaced with the default (-1). To explicitly use CPU-only mode, add `--n-gpu-layers 0` to `extra_args`.

### Extra Arguments

The `extra_args` field accepts a list of strings that are passed directly to llama-server. This allows using any llama-server option, including new options from future llama.cpp releases.

Each element can contain space-separated flag and value pairs for convenience:

```yaml
# Recommended: space-separated format (more readable)
extra_args:
  - "-b 2048"
  - "-ub 2048"
  - "--temp 0.7"
  - "--jinja"
  - "--parallel 1"

# Also supported: separate elements (legacy format)
extra_args:
  - "-b"
  - "2048"
  - "--jinja"

# Mixed format works too
extra_args:
  - "-b 2048"
  - "--jinja"
```

**Limitation:** Values containing spaces are not supported in the space-separated format. For such cases, use the separate elements format or consider alternative approaches (e.g., file-based templates for `--chat-template`).

## Examples

### Basic Preset (File Path)

```yaml
# Absolute path
name: mistral-7b-q4
model: "f:/Users/username/.alpaca/models/mistral-7b.Q4_K_M.gguf"
context_size: 4096
gpu_layers: 35
```

```yaml
# Home directory
name: mistral-7b-q4
model: "f:~/.alpaca/models/mistral-7b.Q4_K_M.gguf"
context_size: 4096
gpu_layers: 35
```

```yaml
# Relative to current working directory
name: codellama
model: "f:./models/codellama.gguf"
context_size: 4096
gpu_layers: 35
```

### Preset with HuggingFace Model Reference

```yaml
# HuggingFace format (auto-resolved at runtime)
name: gemma3-4b-q4
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
context_size: 4096
gpu_layers: 35
```

**Note:** HuggingFace models must be downloaded first with `alpaca pull h:org/repo:quant`. The model field will be automatically resolved to `f:/path/to/downloaded/file.gguf` at runtime.

### Full-Featured Preset

```yaml
name: codellama-34b-instruct
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
name: llama3-8b-network
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
model: "f:./model.gguf"                # Relative to current working directory
model: "f:../shared/model.gguf"        # Parent of current working directory
```

**Important:** Relative paths are resolved from the current working directory when `alpaca load` is executed, NOT from the preset file's directory. It's recommended to use absolute paths or home directory paths (`~/`) for clarity.

The `f:` prefix is stripped when passing the path to llama-server.

### 2. HuggingFace Format (`h:`)

HuggingFace models must use the `h:` prefix:

```yaml
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M"
```

**Resolution process:**
1. Model must be downloaded first with `alpaca pull h:org/repo:quant`
2. At runtime, `h:org/repo:quant` is resolved to `f:/path/to/downloaded/file.gguf`
3. The `f:` prefix is stripped when starting llama-server

**Error handling:**
- Missing prefix → Parse error with clear message
- HuggingFace model not downloaded → Error with suggestion to run `alpaca pull`
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
