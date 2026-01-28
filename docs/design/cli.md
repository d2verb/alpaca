# CLI Commands

## Overview

The `alpaca` CLI communicates with the daemon via Unix socket to manage llama-server.

## Command Reference

### Daemon Management

#### `alpaca start`

Start the Alpaca daemon.

```bash
$ alpaca start
Starting daemon...
Daemon started successfully.
```

If already running:
```bash
$ alpaca start
Daemon is already running.
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
Daemon: running
Model: unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
Status: running
Endpoint: http://localhost:8080
```

When no model is loaded:
```bash
$ alpaca status
Daemon: running
Model: none
Status: idle
```

When daemon is not running:
```bash
$ alpaca status
Daemon: not running
```

### Model Management

#### `alpaca load <preset|repo:quant>`

Load a model using a preset or HuggingFace repository.

**Using preset:**
```bash
$ alpaca load codellama-7b-q4
Loading codellama-7b-q4...
Model loaded. Endpoint: http://localhost:8080
```

**Using HuggingFace format (auto-download if not present):**
```bash
$ alpaca load unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
Model not found. Downloading...
Fetching file list...
Downloading qwen3-coder-30b-a3b-instruct.Q4_K_M.gguf (16 GB)...
[████████████████████████████████] 100%
Loading unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M...
Model loaded. Endpoint: http://localhost:8080
```

If another model is running, it will be stopped first:
```bash
$ alpaca load mistral-7b
Stopping current model...
Loading mistral-7b...
Model loaded. Endpoint: http://localhost:8080
```

**Argument resolution:**
- Contains `:` → HuggingFace format (`<repo>:<quant>`)
- Otherwise → Preset name

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
Stopping model...
Model stopped.
```

### Preset Management

#### `alpaca preset list`

List available presets.

```bash
$ alpaca preset list
codellama-7b-q4
mistral-7b
deepseek-coder
```

With details:
```bash
$ alpaca preset list -v
NAME              MODEL                                                     PORT
codellama-7b-q4   TheBloke/CodeLlama-7B-GGUF:Q4_K_M                         8080
mistral-7b        TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q4_K_M             8080
deepseek-coder    TheBloke/deepseek-coder-6.7B-instruct-GGUF:Q4_K_M        8081
```

#### `alpaca preset rm <name>`

Remove a preset.

```bash
$ alpaca preset rm codellama-7b-q4
Remove preset 'codellama-7b-q4'? [y/N]: y
Removed.
```

### Model File Management

#### `alpaca model list`

List downloaded models.

```bash
$ alpaca model list
unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M  (16 GB)
TheBloke/CodeLlama-7B-GGUF:Q4_K_M            (4.1 GB)
TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M (4.8 GB)
```

Model information is stored in `~/.alpaca/models/.metadata.json`.

#### `alpaca model pull <repo:quant>`

Download a model from HuggingFace.

```bash
$ alpaca model pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M
Fetching file list...
Downloading codellama-7b.Q4_K_M.gguf (4.1 GB)...
[████████████████████████████████] 100%
Saved to: ~/.alpaca/models/codellama-7b.Q4_K_M.gguf
```

**Format**: `<organization>/<repository>:<quantization>`

**Examples**:
```bash
alpaca model pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M
alpaca model pull TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
alpaca model pull unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
```

**Errors**:

Missing quantization type:
```bash
$ alpaca model pull TheBloke/CodeLlama-7B-GGUF
Error: quantization type required (e.g., :Q4_K_M)
```

No matching file:
```bash
$ alpaca model pull TheBloke/CodeLlama-7B-GGUF:Q9_X
Error: no matching file found for 'Q9_X'
Available: Q3_K_M, Q4_K_M, Q5_K_M, Q8_0
```

#### `alpaca model rm <repo:quant>`

Remove a downloaded model.

```bash
$ alpaca model rm unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
Remove unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M (16 GB)? [y/N]: y
Removed.
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
- Created/updated when `alpaca model pull` is run
- Read by `alpaca model list` to display HuggingFace format
- Used by `alpaca load` to resolve `<repo:quant>` to filenames
- Removed when `alpaca model rm` is run

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Daemon not running |
| 3 | Preset/Model not found |
| 4 | Model file not found |
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
