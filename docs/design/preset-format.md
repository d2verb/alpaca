# Preset Format

## Overview

Presets define a model + argument combination that can be loaded with a single command. Alpaca supports two types of presets:

1. **Global presets**: Stored in `~/.alpaca/presets/`, available from anywhere
2. **Local presets**: Stored as `.alpaca.yaml` in a project directory

## File Locations

### Global Presets

```text
~/.alpaca/presets/
├── a1b2c3d4e5f67890.yaml
├── 1234567890abcdef.yaml
└── fedcba0987654321.yaml
```

Global preset files are stored with random filenames (16 hex characters). The `name` field inside the YAML file is used as the identifier for loading (e.g., `alpaca load p:codellama-7b`).

### Local Presets

```text
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

# Optional: draft model for speculative decoding (--model-draft)
draft-model: "f:~/.alpaca/models/codellama-1b-Q4_K_M.gguf"

# Alpaca-level options (optional)
port: 8080              # default: 8080
host: 127.0.0.1         # default: 127.0.0.1

# llama-server options (optional)
# key = llama-server long option name without the -- prefix
options:
  ctx-size: 4096
  threads: 8
  flash-attn: on            # value option: → --flash-attn on
  mlock: true               # boolean flag: → --mlock
  no-mmap: true             # boolean flag: → --no-mmap
```

## Field Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Preset identifier used for loading (e.g., `alpaca load p:name`). Must match `[a-zA-Z0-9_-]+`. |
| `model` | string | Model identifier with explicit prefix: `h:org/repo:quant` (HuggingFace) or `f:/path/to/file` (file path). |

### Optional Fields (Common)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `mode` | string | `"single"` | `"single"` or `"router"` |
| `draft-model` | string | - | Draft model identifier for speculative decoding (`--model-draft`). Uses `f:` or `h:` prefix. |
| `port` | int | 8080 | llama-server listen port |
| `host` | string | `"127.0.0.1"` | llama-server listen host |
| `options` | Options | - | llama-server options (see [Options Map](#options-map)) |

### Options Map

The `options` field is a key-value map for passing arbitrary options to llama-server. Keys are llama-server long option names without the `--` prefix.

```text
llama-server flag    → options key
--ctx-size           → ctx-size
--threads            → threads
--flash-attn         → flash-attn
--cont-batching      → cont-batching
--mlock              → mlock
--temp               → temp
--batch-size         → batch-size
```

#### Value Types

YAML values are written naturally as strings, numbers, or booleans. Internally all values are stored as strings (`map[string]string`) via a custom `UnmarshalYAML` implementation.

go-yaml v3 follows YAML 1.2 where only `true`/`false` are `!!bool`. Values like `on`/`off`/`yes`/`no` are treated as plain strings (`!!str`), so no quoting is needed for value options like `flash-attn: on`.

Boolean case variants (`True`/`TRUE`/`False`/`FALSE`) are normalized to lowercase `"true"`/`"false"` by `UnmarshalYAML`.

```yaml
options:
  ctx-size: 4096        # YAML !!int   → Go "4096"
  temp: 0.7             # YAML !!float → Go "0.7"
  mlock: true           # YAML !!bool  → Go "true"
  mlock: True           # YAML !!bool  → Go "true" (normalized)
  mlock: FALSE          # YAML !!bool  → Go "false" (normalized)
  flash-attn: on        # YAML !!str   → Go "on" (no quoting needed)
  cache-type-k: q8_0    # YAML !!str   → Go "q8_0"
```

#### Single Mode Conversion Rules

`options` map entries are converted to CLI arguments:

| Value | Conversion | Use case | Example |
|-------|-----------|----------|---------|
| `"true"` | `--key` (flag only) | Boolean flags | `mlock: true` → `--mlock` |
| `"false"` | (skipped) | Disable boolean flag | `mlock: false` → (nothing) |
| Other | `--key value` | Value options | `ctx-size: 4096` → `--ctx-size 4096` |

```yaml
# Input
options:
  ctx-size: 4096
  flash-attn: on
  mlock: true
  no-mmap: true
  temp: 0.7

# Generated CLI arguments
# --ctx-size 4096 --flash-attn on --mlock --no-mmap --temp 0.7
```

> **User responsibility**: Alpaca does not manage llama-server flag types (thin wrapper principle). Use `true`/`false` for boolean flags and actual values for value options.

#### Router Mode Conversion Rules

`options` map entries are written directly as `key = value` pairs in config.ini. No special handling of `true`/`false`.

```yaml
# Input
options:
  flash-attn: on
  mlock: true
  cache-type-k: q8_0

# config.ini output
# [*]
# cache-type-k = q8_0
# flash-attn = on
# mlock = true
```

In router mode, all values are written as-is to the INI file. llama-server's INI parser handles type conversion internally (e.g., `mlock = true` is interpreted as `--mlock`).

#### Reserved Keys

The following keys are prohibited in `options` (managed by top-level or dedicated fields):

| Reserved Key | Reason |
|-------------|--------|
| `port` | Conflicts with top-level `port` |
| `host` | Conflicts with top-level `host` |
| `model` | Conflicts with top-level/ModelEntry `model` |
| `model-draft` | Conflicts with top-level/ModelEntry `draft-model` |
| `models-max` | Conflicts with top-level `max-models` (router mode) |
| `sleep-idle-seconds` | Conflicts with top-level `idle-timeout` (router mode) |

## Router Mode

Presets can define multiple models to run simultaneously using llama-server's router mode. This is useful for scenarios like chat + embedding, role-based model assignment, or A/B testing.

### Router Mode Format

```yaml
name: my-workspace
mode: router
port: 8080
host: 127.0.0.1
max-models: 3
idle-timeout: 300

# Global llama-server options ([*] section in config.ini)
options:
  flash-attn: on
  cache-type-k: q8_0

# Model definitions
models:
  - name: qwen3
    model: "h:Qwen/Qwen3-8B-GGUF:Q4_K_M"
    draft-model: "h:Qwen/Qwen3-1B-GGUF:Q4_K_M"
    options:
      ctx-size: 8192
  - name: nomic-embed
    model: "h:nomic-ai/nomic-embed-text-v2-moe-GGUF:Q4_K_M"
    options:
      ctx-size: 2048
      embeddings: true
```

### Router Mode Fields

| Field | Type | Description |
|-------|------|-------------|
| `mode` | string | Must be `"router"` to enable router mode. |
| `max-models` | int | Max simultaneously loaded models (`--models-max`). Omit to use llama-server default. |
| `idle-timeout` | int | Auto-unload after N seconds idle (`--sleep-idle-seconds`). Omit to use llama-server default. |
| `options` | Options | Global llama-server options applied to all models (output as `[*]` section in config.ini). |
| `models` | []ModelEntry | List of models to serve. At least one required. |

### ModelEntry Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Model identifier (section name in config.ini). Must match `[a-zA-Z0-9_-]+`. |
| `model` | string | Model path with `h:` or `f:` prefix. |
| `draft-model` | string | Draft model for speculative decoding (optional). Uses `f:` or `h:` prefix. |
| `options` | Options | Per-model llama-server options (overrides global options). |

### Validation Rules

#### Common

- `name` is required. Must match `[a-zA-Z0-9_-]+`
- `mode` must be `"single"` or `"router"`. Defaults to `"single"` when omitted
- `options` keys and values must not contain newline characters

#### Single Mode

- `model` is required
- `model` value must start with `f:` or `h:` prefix
- `draft-model`, if specified, must start with `f:` or `h:` prefix
- `models`, `max-models`, `idle-timeout` are not allowed
- Reserved keys (`port`, `host`, `model`, `model-draft`, `models-max`, `sleep-idle-seconds`) are not allowed in `options`

#### Router Mode

- `models` is required with at least one entry
- Top-level `model`, `draft-model` are not allowed
- Each ModelEntry `name` is required and must be unique
- Each ModelEntry `model` is required
- Each ModelEntry `draft-model`, if specified, must start with `f:` or `h:` prefix
- Reserved keys (`port`, `host`, `model`, `model-draft`, `models-max`, `sleep-idle-seconds`) are not allowed in top-level `options`
- `port`, `host`, `model`, `model-draft` are not allowed in ModelEntry `options`

## Examples

### Basic Preset (File Path)

```yaml
# Absolute path
name: mistral-7b-q4
model: "f:/Users/username/.alpaca/models/mistral-7b.Q4_K_M.gguf"
```

```yaml
# Home directory
name: mistral-7b-q4
model: "f:~/.alpaca/models/mistral-7b.Q4_K_M.gguf"
```

```yaml
# Relative to current working directory
name: codellama
model: "f:./models/codellama.gguf"
```

### Preset with HuggingFace Model Reference

```yaml
# HuggingFace format (auto-resolved at runtime)
name: gemma3-4b-q4
model: "h:unsloth/gemma3-4b-it-GGUF:Q4_K_M"
```

**Note:** HuggingFace models must be downloaded first with `alpaca pull h:org/repo:quant`. The model field will be automatically resolved to `f:/path/to/downloaded/file.gguf` at runtime.

### Preset with Draft Model (Speculative Decoding)

```yaml
name: llama3-70b-speculative
model: "f:~/.alpaca/models/llama3-70b.Q4_K_M.gguf"
draft-model: "f:~/.alpaca/models/llama3-8b.Q4_K_M.gguf"
options:
  ctx-size: 8192
  threads: 12
```

The `draft-model` field accepts the same format as `model` (`f:` for file paths, `h:` for HuggingFace). The draft model is passed to llama-server via the `--model-draft` flag for speculative decoding.

### Full-Featured Preset

```yaml
name: codellama-34b-instruct
model: "f:~/.alpaca/models/codellama-34b-instruct.Q4_K_M.gguf"
port: 8081
options:
  ctx-size: 8192
  threads: 12
  flash-attn: on
  cont-batching: true
  mlock: true
  no-mmap: true
```

### Preset with Custom Host

```yaml
name: llama3-8b-network
model: "f:~/.alpaca/models/llama3-8b.Q4_K_M.gguf"
host: "0.0.0.0"  # Listen on all interfaces
port: 8080
options:
  ctx-size: 4096
```

### Router Mode

```yaml
name: my-workspace
mode: router
port: 8080
max-models: 3
idle-timeout: 300
options:
  flash-attn: on
  cache-type-k: q8_0
models:
  - name: qwen3
    model: "h:Qwen/Qwen3-8B-GGUF:Q4_K_M"
    draft-model: "h:Qwen/Qwen3-1B-GGUF:Q4_K_M"
    options:
      ctx-size: 8192
  - name: nomic-embed
    model: "h:nomic-ai/nomic-embed-text-v2-moe-GGUF:Q4_K_M"
    options:
      ctx-size: 2048
      embeddings: true
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
- `options` map provides full llama-server compatibility

### Why `options` as `map[string]string`?

- llama-server option names can be used directly as keys, no Alpaca-specific translation needed
- New llama-server options work immediately without Alpaca changes (forward compatibility)
- Same concept works in both single and router modes, reducing learning cost

### Why `draft-model` is a dedicated field (not in `options`)?

- `draft-model` values use `f:`/`h:` prefixes requiring Alpaca-level path resolution (home directory expansion, HuggingFace → file path conversion)
- All other `options` keys are pass-through to llama-server with no processing
- Putting `draft-model` in `options` would break the principle that `options` = pass-through
- `model` and `draft-model` share the same nature (model identifiers) and belong at the same level

### Why `port`/`host` are not in `options`?

- `port`/`host` are Alpaca-level concerns (the daemon needs them to construct endpoints)
- They are referenced internally by Alpaca, not just passed through to llama-server
- Default value management belongs to Alpaca

### Why `true`/`false` handling differs between single and router mode?

llama-server has two flag types: **boolean flags** (no value, e.g., `--mlock`) and **value options** (value required, e.g., `--ctx-size 4096`).

- **Single mode (CLI args)**: Boolean flags don't accept values on CLI. `--mlock true` would cause an error. So `mlock: true` → `--mlock` (flag only), `mlock: false` → (skipped).
- **Router mode (config.ini)**: All flags use `key = value` format. llama-server's INI parser handles type conversion internally. Values are written as-is.

This difference comes from llama-server's CLI vs INI behavior, not from Alpaca's design. Alpaca absorbs this difference so users can write the same `options` syntax in both modes.

### Extensibility

The `options` map ensures forward compatibility. When llama.cpp adds new options, users can immediately use them without waiting for Alpaca updates.
