# CLI Commands

## Overview

The `alpaca` CLI communicates with the daemon via Unix socket to manage llama-server.

## Command Reference

### Daemon Management

#### `alpaca start`

Start the Alpaca daemon in the background.

```bash
$ alpaca start
Daemon started (PID: 12345)
Logs: /Users/username/.alpaca/logs/daemon.log
```

If already running:
```bash
$ alpaca start
Daemon is already running (PID: 12345).
```

**Flags:**
- `--foreground`, `-f`: Run in foreground (don't daemonize). Useful for debugging.

```bash
$ alpaca start --foreground
# Runs daemon in foreground, logs to stdout
```

#### `alpaca stop`

Stop the Alpaca daemon.

```bash
$ alpaca stop
Stopping daemon...
Daemon stopped.
```

This will also stop any running llama-server process.

#### `alpaca status`

Show current status.

```bash
$ alpaca status
Status: running
Preset: qwen3-coder-30b
Endpoint: http://localhost:8080
Logs: /Users/username/.alpaca/logs/
```

When no model is loaded:
```bash
$ alpaca status
Status: idle
Logs: /Users/username/.alpaca/logs/
```

When daemon is not running:
```bash
$ alpaca status
Daemon is not running.
Run: alpaca start
```

### Model Management

#### `alpaca load <identifier>`

Load a model using an explicit identifier with prefix.

**Identifier Format:**
All identifiers must use an explicit prefix:
- `h:org/repo:quant` - HuggingFace model (auto-download if not present)
- `p:preset-name` - Preset
- `f:/path/to/file` - File path (uses default settings)

**Using preset:**
```bash
$ alpaca load p:codellama-7b-q4
Loading p:codellama-7b-q4...
Model ready at http://localhost:8080
```

**Using HuggingFace format (auto-download if not present):**
```bash
$ alpaca load h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
Model not found. Downloading...
Fetching file list...
Downloading qwen3-coder-30b-a3b-instruct.Q4_K_M.gguf (16.0 GB)...
[████████████████████████████████████████] 100.0% (16.0 GB / 16.0 GB)
Saved to: /Users/username/.alpaca/models/qwen3-coder-30b-a3b-instruct.Q4_K_M.gguf
Loading h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M...
Model ready at http://localhost:8080
```

**Using file path (with default settings):**
```bash
$ alpaca load f:~/models/my-model.gguf
Loading f:~/models/my-model.gguf...
Model ready at http://localhost:8080

$ alpaca load f:./model.gguf
Loading f:./model.gguf...
Model ready at http://localhost:8080
```

File paths are loaded with default settings from `~/.alpaca/config.yaml`:
- `default_host`: 127.0.0.1
- `default_port`: 8080
- `default_ctx_size`: 4096
- `default_gpu_layers`: -1

For custom settings, create a preset instead.

**Error handling:**
```bash
# Missing prefix
$ alpaca load my-preset
Error: invalid identifier format 'my-preset'
Expected: h:org/repo:quant, p:preset-name, or f:/path/to/file
Examples: alpaca load p:my-preset

# Missing quant in HuggingFace
$ alpaca load h:unsloth/gemma3
Error: missing quant specifier in HuggingFace identifier
Expected format: h:org/repo:quant (e.g., h:unsloth/gemma3:Q4_K_M)
```

If another model is running, it will be stopped first automatically

**Default settings for HuggingFace models:**
When loading a model without a preset, the following defaults are used:
```yaml
host: 127.0.0.1
port: 8080
ctx_size: 4096
n_gpu_layers: -1  # Use all GPU layers
```

These can be overridden in `~/.alpaca/config.yaml`:
```yaml
llama_server_path: llama-server
default_port: 8080
default_host: 127.0.0.1
default_ctx_size: 4096
default_gpu_layers: -1
```

#### `alpaca unload`

Stop the currently running model.

```bash
$ alpaca unload
Model stopped.
```

### Preset Management

#### `alpaca preset list` (or `alpaca preset ls`)

List available presets.

```bash
$ alpaca preset list
Available presets:
  - codellama-7b-q4
  - mistral-7b
  - deepseek-coder
```

When no presets exist:
```bash
$ alpaca preset list
No presets available.
Add presets to: /Users/username/.alpaca/presets
```

#### `alpaca preset rm <name>`

Remove a preset.

```bash
$ alpaca preset rm codellama-7b-q4
Delete preset 'codellama-7b-q4'? (y/N): y
Preset 'codellama-7b-q4' removed.
```

If preset doesn't exist:
```bash
$ alpaca preset rm nonexistent
Preset 'nonexistent' not found.
```

### Model File Management

#### `alpaca model list` (or `alpaca model ls`)

List downloaded models.

```bash
$ alpaca model list
Downloaded models:
  - h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M (16.0 GB)
  - h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M (4.1 GB)
  - h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M (4.8 GB)
```

When no models are downloaded:
```bash
$ alpaca model list
No models downloaded.
Run: alpaca model pull h:org/repo:quant
```

Model information is stored in `~/.alpaca/models/.metadata.json`.

#### `alpaca model pull h:org/repo:quant`

Download a model from HuggingFace.

```bash
$ alpaca model pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
Fetching file list...
Downloading codellama-7b.Q4_K_M.gguf (4.1 GB)...
[████████████████████████████████████████] 100.0% (4.1 GB / 4.1 GB)
Saved to: /Users/username/.alpaca/models/codellama-7b.Q4_K_M.gguf
```

**Format**: `h:<organization>/<repository>:<quantization>`

**Examples**:
```bash
alpaca model pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
alpaca model pull h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
alpaca model pull h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
```

**Errors**:

Missing h: prefix:
```bash
$ alpaca model pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M
Error: model pull only supports HuggingFace models
Format: alpaca model pull h:org/repo:quant
Example: alpaca model pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```

Missing quant:
```bash
$ alpaca model pull h:TheBloke/CodeLlama-7B-GGUF
Error: missing quant specifier
Format: alpaca model pull h:org/repo:quant
Example: alpaca model pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```

#### `alpaca model rm h:org/repo:quant`

Remove a downloaded model.

```bash
$ alpaca model rm h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
Delete model 'h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M'? (y/N): y
Model 'h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M' removed.
```

If model doesn't exist:
```bash
$ alpaca model rm h:nonexistent:Q4_K_M
Model 'h:nonexistent:Q4_K_M' not found.
```

This removes both the model file and its metadata entry.

## Metadata Management

Model metadata is stored in `~/.alpaca/models/.metadata.json`:

```json
{
  "models": [
    {
      "repo": "unsloth/qwen3-coder-30b-a3b-instruct",
      "quant": "Q4_K_M",
      "filename": "qwen3-coder-30b-a3b-instruct.Q4_K_M.gguf",
      "size": 17179869184,
      "downloaded_at": "2026-01-28T10:30:00Z"
    }
  ]
}
```

This metadata is:
- Created/updated when `alpaca model pull h:<repo>:<quant>` is run
- Read by `alpaca model list` to display HuggingFace format
- Used by `alpaca load h:<repo>:<quant>` to resolve identifiers to filenames
- Removed when `alpaca model rm h:<repo>:<quant>` is run

## Daemon Behavior

The daemon runs in the background by default:
- Logs to `~/.alpaca/logs/daemon.log` (daemon operations)
- Logs to `~/.alpaca/logs/llama.log` (llama-server output)
- Unix socket at `~/.alpaca/alpaca.sock`
- PID file at `~/.alpaca/alpaca.pid`

Logs are rotated automatically (50MB max size, 3 backups, 7 days retention, gzip compressed).

Use `--foreground` flag to run in foreground for debugging.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Daemon not running |
| 3 | Preset not found |
| 4 | Model not found |
| 5 | Download failed |

## Global Flags

| Flag | Description |
|------|-------------|
| `--help`, `-h` | Show help |
| `--version`, `-v` | Show version |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ALPACA_HOME` | Alpaca home directory | `~/.alpaca` |
| `ALPACA_SOCKET` | Unix socket path | `$ALPACA_HOME/alpaca.sock` |
