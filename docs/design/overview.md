# Alpaca - Overview

## What is Alpaca?

Alpaca is a lightweight wrapper around `llama-server` (from llama.cpp) for macOS. It provides both CLI and GUI interfaces, similar to Ollama, but with full access to llama-server's options and better performance.

## Motivation

1. **Too many arguments**: llama-server has numerous command-line arguments. Specifying them every time is tedious.

2. **Model management is difficult**: Especially for models downloaded from Hugging Face, it's hard to track where they are stored.

3. **Model switching requires restart**: To switch models, you need to stop and restart llama-server manually.

4. **No preset support**: There's no way to save successful configurations (model + arguments), making it hard to reproduce good setups.

5. **Ollama limitations**: While Ollama is convenient, it has:
   - Worse performance compared to raw llama.cpp
   - Slower adoption of llama.cpp updates
   - Fewer options than raw llama.cpp

## Goals

- Thin wrapper around llama-server (proxy tool approach)
- CLI + GUI (macOS menu bar app)
- Preset system for model + argument combinations
- Smooth model switching experience
- Full access to llama-server options
- Simple model management

## Non-Goals

- Building our own inference engine
- Managing llama.cpp versions (use system-installed llama-server)
- Complex health monitoring (rely on llama-server's /health endpoint)

## Target Platform

- macOS (primary)
- Future: Windows, Linux (CLI/Daemon are cross-platform by design)
