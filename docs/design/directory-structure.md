# Directory Structure

## Alpaca Home Directory

All Alpaca data is stored under `~/.alpaca/`:

```
~/.alpaca/
├── alpaca.sock          # Unix socket for daemon communication
├── alpaca.pid           # Daemon PID file
├── router-config.ini    # Router mode config (generated at runtime)
├── presets/             # Preset definitions (random filenames)
│   ├── a1b2c3d4e5f67890.yaml
│   ├── 1234567890abcdef.yaml
│   └── ...
├── models/              # Downloaded models
│   ├── .metadata.json   # Model download metadata
│   ├── codellama-7b-Q4_K_M.gguf
│   ├── mistral-7b-instruct-v0.2.Q4_K_M.gguf
│   └── ...
└── logs/                # Log files (created automatically)
    ├── daemon.log       # Daemon process logs
    └── llama.log        # llama-server output logs
```

## Files

### alpaca.sock

Unix socket file for communication between CLI/GUI and daemon.

- Created when daemon starts
- Removed when daemon stops
- Permissions: 0600 (owner only)

### alpaca.pid

Contains the PID of the running daemon process.

- Used to check if daemon is running
- Used to send signals to daemon

### router-config.ini

Generated config file for router mode. Written atomically (temp file + rename) when loading a router preset, and cleaned up on model stop (best-effort).

## Directories

### presets/

Contains preset YAML files. Each file defines a model + argument combination.

See [preset-format.md](./preset-format.md) for details.

### models/

Default location for downloaded models.

- `alpaca pull` downloads models here
- Presets can reference models here or anywhere else on the filesystem
- `.metadata.json`: Tracks downloaded models (repo, quant, filename, size, download date)

### logs/

Log files for debugging. Created automatically when daemon starts.

- `daemon.log`: Daemon process logs (startup, shutdown, errors)
- `llama.log`: llama-server stdout/stderr output

**Rotation Policy:**
- Max size: 50MB per file
- Max backups: 3 old files kept
- Max age: 7 days
- Compression: Enabled (old logs are gzipped)

## Model Storage

### Default Behavior

Models downloaded via `alpaca pull` are stored in `~/.alpaca/models/`.

```
$ alpaca pull h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M
Downloading codellama-7b-Q4_K_M.gguf...
Saved to: ~/.alpaca/models/codellama-7b-Q4_K_M.gguf
```

### Custom Model Paths

Presets can reference models anywhere on the filesystem:

```yaml
# Using default location
model: "f:~/.alpaca/models/codellama-7b-Q4_K_M.gguf"

# Using custom location
model: "f:/Volumes/ExternalDrive/models/llama3-70b.gguf"

# Using home directory path
model: "f:~/Downloads/some-model.gguf"

# Using HuggingFace format (resolved at runtime)
model: "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M"
```

## Initialization

On first run, Alpaca creates the directory structure automatically:

```
$ alpaca start
Creating ~/.alpaca/
Creating ~/.alpaca/presets/
Creating ~/.alpaca/models/
Creating ~/.alpaca/logs/
Starting daemon...
```

**Created directories:**
- `~/.alpaca/` (home directory)
- `~/.alpaca/presets/` (preset definitions)
- `~/.alpaca/models/` (downloaded models)
- `~/.alpaca/logs/` (daemon and llama-server logs)

All directories are created with `0755` permissions. If directories already exist, they are left untouched.
