# CLI Commands

## Overview

The `alpaca` CLI communicates with the daemon via Unix socket to manage llama-server.

## Command Reference

### Daemon Management

#### `alpaca start`

Start the Alpaca daemon in the background.

```bash
$ alpaca start
✓ Daemon started (PID: 12345)
ℹ Logs: /Users/username/.alpaca/logs/daemon.log
```

If already running:
```bash
$ alpaca start
ℹ Daemon is already running (PID: 12345).
```

There is no foreground mode. The daemon always runs in the background.

#### `alpaca stop`

Stop the Alpaca daemon.

```bash
$ alpaca stop
ℹ Stopping daemon...
✓ Daemon stopped.
```

This will also stop any running llama-server process.

#### `alpaca status`

Show current status.

```bash
$ alpaca status
🚀 Status
  State          ● Running
  Preset         p:qwen3-coder-30b
  Endpoint       http://localhost:8080
  Logs           /Users/username/.alpaca/logs/llama.log
```

When no model is loaded:
```bash
$ alpaca status
🚀 Status
  State          ○ Idle
  Logs           /Users/username/.alpaca/logs/llama.log
```

When daemon is not running:
```bash
$ alpaca status
✗ Daemon is not running.
ℹ Run: alpaca start
```

#### `alpaca open`

Open the llama-server endpoint in your default browser.

```bash
$ alpaca open
ℹ Opening http://127.0.0.1:8080 in browser...
```

When no model is loaded:
```bash
$ alpaca open
✗ Server is not running.
ℹ Run: alpaca load <preset>
```

When daemon is not running:
```bash
$ alpaca open
✗ Daemon is not running.
ℹ Run: alpaca start
```

#### `alpaca logs`

View daemon or llama-server logs.

```bash
# View daemon logs (default)
$ alpaca logs
[2026-01-29 10:30:00] INFO daemon starting
[2026-01-29 10:30:01] INFO server listening on /Users/username/.alpaca/alpaca.sock
[2026-01-29 10:30:15] INFO loading model p:codellama-7b
```

**Flags:**
- `-f, --follow`: Follow log output in real-time (like `tail -f`)
- `-d, --daemon`: Show daemon logs (default)
- `-s, --server`: Show llama-server logs

**Examples:**

Follow daemon logs in real-time:
```bash
$ alpaca logs --follow
# or
$ alpaca logs -f
```

View llama-server output:
```bash
$ alpaca logs --server
# or
$ alpaca logs -s
```

Follow llama-server logs:
```bash
$ alpaca logs -f -s
```

**Note:** This command uses `/usr/bin/tail` under the hood. Log files are located at:
- Daemon: `~/.alpaca/logs/daemon.log`
- llama-server: `~/.alpaca/logs/llama.log`

### Model Management

#### `alpaca load [identifier]`

Load a model using an explicit identifier with prefix, or load a local preset.

**Identifier Format:**
All identifiers must use an explicit prefix:
- `h:org/repo:quant` - HuggingFace model (auto-download if not present)
- `p:preset-name` - Global preset
- `f:/path/to/file` - File path (uses default settings)
- `f:*.yaml` or `f:*.yml` - Local preset file

**No argument (local preset):**
When run without arguments, loads `.alpaca.yaml` from the current directory:
```bash
$ cd my-project
$ alpaca load
ℹ Loading my-project...
✓ Model ready at http://localhost:8080
```

If no `.alpaca.yaml` exists:
```bash
$ alpaca load
✗ Error: no .alpaca.yaml found in current directory
ℹ Run: alpaca new --local
```

**Using preset:**
```bash
$ alpaca load p:codellama-7b-q4
ℹ Loading p:codellama-7b-q4...
✓ Model ready at http://localhost:8080
```

**Using HuggingFace format (auto-download if not present):**
```bash
$ alpaca load h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
ℹ Model not found. Downloading...
ℹ Fetching file list...
ℹ Downloading qwen3-coder-30b-a3b-instruct.Q4_K_M.gguf (16.0 GB)...
[████████████████████████████████████████] 100.0% (16.0 GB / 16.0 GB)
✓ Saved to: /Users/username/.alpaca/models/qwen3-coder-30b-a3b-instruct.Q4_K_M.gguf
ℹ Loading h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M...
✓ Model ready at http://localhost:8080
```

**Using file path (with default settings):**
```bash
$ alpaca load f:~/models/my-model.gguf
ℹ Loading f:~/models/my-model.gguf...
✓ Model ready at http://localhost:8080

$ alpaca load f:./model.gguf
ℹ Loading f:./model.gguf...
✓ Model ready at http://localhost:8080
```

**Using local preset file:**
```bash
$ alpaca load f:./custom-preset.yaml
ℹ Loading my-custom-preset...
✓ Model ready at http://localhost:8080

$ alpaca load f:../shared/preset.yaml
ℹ Loading shared-preset...
✓ Model ready at http://localhost:8080
```

File paths are loaded with default settings:
- `host`: 127.0.0.1
- `port`: 8080
- `context_size`: 4096

**Error handling:**
```bash
# Missing prefix
$ alpaca load my-preset
✗ Error: invalid identifier format 'my-preset'
ℹ Expected: h:org/repo:quant, p:preset-name, or f:/path/to/file
ℹ Examples: alpaca load p:my-preset

# Missing quant in HuggingFace
$ alpaca load h:unsloth/gemma3
✗ Error: missing quant specifier in HuggingFace identifier
ℹ Expected format: h:org/repo:quant (e.g., h:unsloth/gemma3:Q4_K_M)
```

If another model is running, it will be stopped first automatically

**Default settings for HuggingFace models:**
When loading a model without a preset, the following defaults are used:
```yaml
host: 127.0.0.1
port: 8080
context_size: 4096
```

These defaults are defined in the preset package constants.

#### `alpaca unload`

Stop the currently running model.

```bash
$ alpaca unload
✓ Model stopped.
```

### Preset Management

#### `alpaca ls`

List available presets and downloaded models.

```bash
$ alpaca ls
📦 Presets
──────────
  p:codellama-7b-q4
  p:mistral-7b
  p:deepseek-coder

🤖 Models
─────────
  h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
    4.1 GB · Downloaded 2024-01-15
  h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
    4.8 GB · Downloaded 2024-01-14
```

When no presets or models exist:
```bash
$ alpaca ls
No presets available.

  Create one:  alpaca new

No models downloaded.

  Download one:  alpaca pull h:org/repo:quant
```

#### `alpaca show <identifier>`

Show detailed information for a preset or model.

**Show preset details:**
```bash
$ alpaca show p:codellama-7b-q4
📦 Preset: p:codellama-7b-q4
  Model          f:/Users/username/.alpaca/models/codellama-7b.Q4_K_M.gguf
  Context Size   4096
  Endpoint       127.0.0.1:8080
```

**Show model details:**
```bash
$ alpaca show h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
🤖 Model: h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
  Filename       codellama-7b.Q4_K_M.gguf
  Size           4.1 GB
  Downloaded     2026-01-28 10:30:00
  Path           /Users/username/.alpaca/models/codellama-7b.Q4_K_M.gguf
  Status         ✓ Ready
```

**Error cases:**

If preset doesn't exist:
```bash
$ alpaca show p:nonexistent
✗ Preset 'nonexistent' not found.
```

If model not downloaded:
```bash
$ alpaca show h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
✗ Model 'h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M' not downloaded
ℹ Run: alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```

File paths cannot be shown:
```bash
$ alpaca show f:/path/to/model.gguf
✗ Error: cannot show file details
ℹ Use: alpaca show p:name or alpaca show h:org/repo:quant
```

#### `alpaca new`

Create a new preset interactively.

**Global preset (default):**
```bash
$ alpaca new
📦 Create Preset
Name: my-model
Model: h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
Host [127.0.0.1]:
Port [8080]:
Context [2048]: 8192
✓ Created 'my-model'
💡 alpaca load p:my-model
```

**Flags:**
- `--local`: Create `.alpaca.yaml` in the current directory instead of a global preset

**Local preset:**
```bash
$ cd my-project
$ alpaca new --local
📦 Create Local Preset
Name [my-project]:
Model: f:./models/model.gguf
Host [127.0.0.1]:
Port [8080]:
Context [2048]: 4096
✓ Created '.alpaca.yaml'
💡 alpaca load
```

When using `--local`:
- The default name is derived from the current directory name
- The preset is saved to `.alpaca.yaml` in the current directory
- Relative paths in the model field are supported and resolved from the preset file's directory

The command will prompt for:
- **Name**: Name for the preset file (without .yaml extension) - **required**
- **Model**: Model identifier (must include `f:` or `h:` prefix) - **required**
- **Host**: Server host address (default: 127.0.0.1)
- **Port**: Server port (default: 8080)
- **Context**: Context window size (default: 2048)

Press Enter to accept default values (shown in brackets). Only non-default values are written to the YAML file.

Additional settings (threads, extra_args) can be added by editing the generated YAML file.

#### `alpaca rm p:<name>`

Remove a preset.

```bash
$ alpaca rm p:codellama-7b-q4
Delete preset 'codellama-7b-q4'? (y/N): y
✓ Preset 'codellama-7b-q4' removed.
```

If preset doesn't exist:
```bash
$ alpaca rm p:nonexistent
✗ Preset 'nonexistent' not found.
```

### Model File Management

See `alpaca ls` above for listing models.

Model information is stored in `~/.alpaca/models/.metadata.json`.

#### `alpaca pull h:org/repo:quant`

Download a model from HuggingFace.

```bash
$ alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
ℹ Fetching file list...
ℹ Downloading codellama-7b.Q4_K_M.gguf (4.1 GB)...
[████████████████████████████████████████] 100.0% (4.1 GB / 4.1 GB)
✓ Saved to: /Users/username/.alpaca/models/codellama-7b.Q4_K_M.gguf
```

**Format**: `h:<organization>/<repository>:<quantization>`

**Examples**:
```bash
alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
alpaca pull h:TheBloke/Mistral-7B-Instruct-v0.2-GGUF:Q5_K_M
alpaca pull h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
```

**Errors**:

Missing h: prefix:
```bash
$ alpaca pull TheBloke/CodeLlama-7B-GGUF:Q4_K_M
✗ Error: pull only supports HuggingFace models
ℹ Format: alpaca pull h:org/repo:quant
ℹ Example: alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```

Missing quant:
```bash
$ alpaca pull h:TheBloke/CodeLlama-7B-GGUF
✗ Error: missing quant specifier
ℹ Format: alpaca pull h:org/repo:quant
ℹ Example: alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
```

#### `alpaca rm h:org/repo:quant`

Remove a downloaded model.

```bash
$ alpaca rm h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M
Delete model 'h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M'? (y/N): y
✓ Model 'h:unsloth/qwen3-coder-30b-a3b-instruct:Q4_K_M' removed.
```

If model doesn't exist:
```bash
$ alpaca rm h:nonexistent:Q4_K_M
✗ Model 'h:nonexistent:Q4_K_M' not found.
```

This removes both the model file and its metadata entry.

## Daemon Behavior

The daemon runs in the background by default:
- Logs to `~/.alpaca/logs/daemon.log` (daemon operations)
- Logs to `~/.alpaca/logs/llama.log` (llama-server output)
- Unix socket at `~/.alpaca/alpaca.sock`
- PID file at `~/.alpaca/alpaca.pid`

Logs are rotated automatically (50MB max size, 3 backups, 7 days retention, gzip compressed).

Foreground mode is not supported.

## Other Commands

### `alpaca version`

Show version information.

```bash
$ alpaca version
alpaca version 0.1.0 (a1b2c3d)
```

The output includes the version number and commit hash for debugging purposes.

### `alpaca upgrade`

Upgrade alpaca to the latest version.

```bash
$ alpaca upgrade
ℹ Checking for updates...

  Current: 0.1.0
  Latest:  0.2.0

ℹ Downloading...
✓ Upgraded to 0.2.0
```

When already up to date:
```bash
$ alpaca upgrade
ℹ Checking for updates...

  Current: 0.2.0
  Latest:  0.2.0

✓ Already up to date
```

**Flags:**
- `--check`, `-c`: Check for updates without installing
- `--force`, `-f`: Force upgrade even if installation source is unknown or mismatched

**Check only mode:**
```bash
$ alpaca upgrade --check
ℹ Checking for updates...

  Current: 0.1.0
  Latest:  0.2.0

ℹ Update available. Run: alpaca upgrade
```

**Installation source detection:**

The upgrade command detects how alpaca was installed and provides appropriate guidance:

- **Homebrew**: Prompts to use `brew upgrade alpaca`
- **apt**: Prompts to use `sudo apt update && sudo apt upgrade alpaca`
- **go install**: Prompts to use `go install github.com/d2verb/alpaca/cmd/alpaca@latest`
- **Install script**: Performs self-update automatically
- **Unknown**: Suggests using `--force` or original installation method

```bash
# Homebrew installation
$ alpaca upgrade
ℹ Installed via Homebrew.
To upgrade, run:

    brew upgrade alpaca
```

### `alpaca completion-script`

Output shell completion script for bash, zsh, or fish.

```bash
# Add to your shell config
# bash
echo "$(alpaca completion-script)" >> ~/.bashrc

# zsh
echo "$(alpaca completion-script)" >> ~/.zshrc

# fish
alpaca completion-script >> ~/.config/fish/config.fish
```

After adding, restart your shell or source the configuration file.

**Supported shells:**
- bash
- zsh
- fish

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
| `--help`, `-h` | Show help for any command |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ALPACA_HOME` | Alpaca home directory | `~/.alpaca` |
| `ALPACA_SOCKET` | Unix socket path | `$ALPACA_HOME/alpaca.sock` |
