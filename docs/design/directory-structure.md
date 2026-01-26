# Directory Structure

## Alpaca Home Directory

All Alpaca data is stored under `~/.alpaca/`:

```
~/.alpaca/
├── config.yaml          # Global configuration
├── alpaca.sock          # Unix socket for daemon communication
├── alpaca.pid           # Daemon PID file
├── presets/             # Preset definitions
│   ├── codellama-7b.yaml
│   ├── mistral-7b.yaml
│   └── ...
├── models/              # Downloaded models
│   ├── codellama-7b-Q4_K_M.gguf
│   ├── mistral-7b-instruct-v0.2.Q4_K_M.gguf
│   └── ...
└── logs/                # Log files (optional)
    └── alpacad.log
```

## Files

### config.yaml

Global configuration for Alpaca.

```yaml
# Path to llama-server binary
llama_server_path: /usr/local/bin/llama-server

# Default port (can be overridden in presets)
default_port: 8080

# Default host (can be overridden in presets)
default_host: "127.0.0.1"
```

### alpaca.sock

Unix socket file for communication between CLI/GUI and daemon.

- Created when daemon starts
- Removed when daemon stops
- Permissions: 0600 (owner only)

### alpaca.pid

Contains the PID of the running daemon process.

- Used to check if daemon is running
- Used to send signals to daemon

## Directories

### presets/

Contains preset YAML files. Each file defines a model + argument combination.

See [preset-format.md](./preset-format.md) for details.

### models/

Default location for downloaded models.

- `alpaca pull` downloads models here
- Presets can reference models here or anywhere else on the filesystem

### logs/ (optional)

Log files for debugging.

- `alpacad.log`: Daemon logs
- Rotation policy: TBD

## Model Storage

### Default Behavior

Models downloaded via `alpaca pull` are stored in `~/.alpaca/models/`.

```
$ alpaca pull TheBloke/CodeLlama-7B-GGUF
Downloading codellama-7b-Q4_K_M.gguf...
Saved to: ~/.alpaca/models/codellama-7b-Q4_K_M.gguf
```

### Custom Model Paths

Presets can reference models anywhere on the filesystem:

```yaml
# Using default location
model: ~/.alpaca/models/codellama-7b-Q4_K_M.gguf

# Using custom location
model: /Volumes/ExternalDrive/models/llama3-70b.gguf

# Using relative path (relative to home)
model: ~/Downloads/some-model.gguf
```

## Initialization

On first run, Alpaca creates the directory structure:

```
$ alpaca start
Creating ~/.alpaca/
Creating ~/.alpaca/presets/
Creating ~/.alpaca/models/
Starting daemon...
```

If directories already exist, they are left untouched.
