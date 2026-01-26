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
Model: codellama-7b-q4
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

#### `alpaca run <preset>`

Load a model using the specified preset.

```bash
$ alpaca run codellama-7b-q4
Loading codellama-7b-q4...
Model loaded. Endpoint: http://localhost:8080
```

If another model is running, it will be stopped first:
```bash
$ alpaca run mistral-7b
Stopping current model...
Loading mistral-7b...
Model loaded. Endpoint: http://localhost:8080
```

#### `alpaca kill`

Stop the currently running model.

```bash
$ alpaca kill
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
NAME              MODEL                                      PORT
codellama-7b-q4   ~/.alpaca/models/codellama-7b.Q4_K_M.gguf  8080
mistral-7b        ~/.alpaca/models/mistral-7b.Q4_K_M.gguf    8080
deepseek-coder    ~/.alpaca/models/deepseek-coder.Q4_K_M.gguf 8081
```

### Model Download

#### `alpaca pull <repo>:<quant>`

Download a model from HuggingFace.

```bash
$ alpaca pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M
Fetching file list...
Downloading codellama-7b.Q4_K_M.gguf (4.1 GB)...
[████████████████████████████████] 100%
Saved to: ~/.alpaca/models/codellama-7b.Q4_K_M.gguf
```

**Format**: `<organization>/<repository>:<quantization>`

**Examples**:
```bash
alpaca pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M
alpaca pull TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
alpaca pull TheBloke/deepseek-coder-6.7B-instruct-GGUF:Q4_K_M
```

**Errors**:

Missing quantization type:
```bash
$ alpaca pull TheBloke/CodeLlama-7B-GGUF
Error: quantization type required (e.g., :Q4_K_M)
```

No matching file:
```bash
$ alpaca pull TheBloke/CodeLlama-7B-GGUF:Q9_X
Error: no matching file found for 'Q9_X'
Available: Q3_K_M, Q4_K_M, Q5_K_M, Q8_0
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Daemon not running |
| 3 | Preset not found |
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
