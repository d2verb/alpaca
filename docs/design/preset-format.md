# Preset Format

## Overview

Presets define a model + argument combination that can be loaded with a single command. Alpaca supports two types of presets:

1. **Global presets**: Stored in `~/.alpaca/presets/`, available from anywhere
2. **Local presets**: Stored as `.alpaca.yaml` in a project directory

## File Locations

### Global Presets

```
~/.alpaca/presets/
├── a1b2c3d4e5f67890.yaml
├── 1234567890abcdef.yaml
└── fedcba0987654321.yaml
```

Global preset files are stored with random filenames (16 hex characters). The `name` field inside the YAML file is used as the identifier for loading (e.g., `alpaca load p:codellama-7b`).

### Local Presets

```
my-project/
├── .alpaca.yaml     # Local preset
├── src/
└── ...
```

Local presets are project-specific configuration files. When you run `alpaca load` without arguments in a directory containing `.alpaca.yaml`, that preset is loaded automatically.

## Format

```yaml
# Required: preset identifier (alphanumeric, underscore, hyphen only)
name: codellama-7b

# Required: model identifier with explicit prefix
model: "f:~/.alpaca/models/codellama-7b-Q4_K_M.gguf"

# Common options (mapped to llama-server arguments)
context_size: 4096      # --ctx-size
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
| `threads` | int | 0 (omit flag, llama-server decides) | `--threads` |
| `port` | int | 8080 | `--port` |
| `host` | string | "127.0.0.1" | `--host` |

**Note on defaults:** Alpaca applies explicit defaults for `context_size`, `host`, and `port`. When these fields are omitted from the YAML, the defaults are still passed to llama-server. `threads` omits the flag when not specified, allowing llama-server to decide.

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
```

```yaml
# Home directory
name: mistral-7b-q4
model: "f:~/.alpaca/models/mistral-7b.Q4_K_M.gguf"
context_size: 4096
```

```yaml
# Relative to current working directory
name: codellama
model: "f:./models/codellama.gguf"
context_size: 4096
```

### Preset with HuggingFace Model Reference

```yaml
# HuggingFace format (auto-resolved at runtime)
name: gemma3-4b-q4
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
context_size: 4096
```

**Note:** HuggingFace models must be downloaded first with `alpaca pull h:org/repo:quant`. The model field will be automatically resolved to `f:/path/to/downloaded/file.gguf` at runtime.

### Full-Featured Preset

```yaml
name: codellama-34b-instruct
model: "f:~/.alpaca/models/codellama-34b-instruct.Q4_K_M.gguf"
context_size: 8192
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

**Path resolution:** For both local and global presets, relative paths (`./` and `../`) are resolved from the preset file's directory. This allows you to reference models relative to your preset location:

```yaml
# In /path/to/project/.alpaca.yaml
model: "f:./models/local-model.gguf"  # Resolves to /path/to/project/models/local-model.gguf
```

**Recommendation for global presets:** Since global presets are stored in `~/.alpaca/presets/`, using relative paths would resolve from that directory (e.g., `f:./model.gguf` → `~/.alpaca/presets/model.gguf`). It's recommended to use absolute paths or home directory paths (`~/`) for global presets.

The `f:` prefix is stripped when passing the path to llama-server.

### 2. HuggingFace Format (`h:`)

HuggingFace models must use the `h:` prefix:

```yaml
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M"
```

**Resolution process:**
1. Model must be downloaded first with `alpaca pull h:org/repo:quant`
2. At load time, the daemon resolves `h:org/repo:quant` to `f:/path/to/downloaded/file.gguf`
3. The `f:` prefix is stripped when starting llama-server

**Note:** The HuggingFace to file path resolution happens in the daemon layer (not the preset package). The preset package stores the model identifier as-is; the daemon looks up the downloaded file path when loading the model.

**Error handling:**
- Missing prefix → Parse error with clear message
- HuggingFace model not downloaded → Error with suggestion to run `alpaca pull`
- File path doesn't exist → Error when starting llama-server

## Local Presets

Local presets allow per-project model configuration using `.alpaca.yaml` files.

### Creating a Local Preset

```bash
$ cd my-project
$ alpaca new --local
📦 Create Local Preset
Name [my-project]:
Model: f:./models/my-model.gguf
Host [127.0.0.1]:
Port [8080]:
Context [2048]: 4096
✓ Created '.alpaca.yaml'
💡 alpaca load
```

The default name is derived from the directory name (sanitized to valid characters).

### Loading a Local Preset

```bash
$ cd my-project
$ alpaca load
ℹ Loading my-project...
✓ Model ready at http://localhost:8080
```

When `alpaca load` is run without arguments, it looks for `.alpaca.yaml` in the current directory.

### Loading from a Specific Path

```bash
$ alpaca load f:./custom-preset.yaml
$ alpaca load f:../shared/preset.yaml
```

Any `.yaml` or `.yml` file can be loaded with the `f:` prefix.

### Relative Paths in Local Presets

Relative paths in the `model` field are resolved from the preset file's directory:

```yaml
# In /path/to/project/.alpaca.yaml
name: my-project
model: "f:./models/model.gguf"      # → /path/to/project/models/model.gguf
model: "f:../shared/model.gguf"     # → /path/to/shared/model.gguf
```

This makes local presets portable - they work correctly regardless of which directory you run `alpaca load` from (as long as you're in the project directory).

### Use Cases

- **Project-specific models**: Different projects using different model configurations
- **Team sharing**: Commit `.alpaca.yaml` to version control for consistent team setup
- **Local development**: Reference models stored relative to the project

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
