# Configuration File Templates

Templates for project setup. Copy these to the project root when initializing.

## Taskfile.yml

```yaml
version: '3'

vars:
  BINARY_NAME: alpaca
  BUILD_DIR: ./build

tasks:
  default:
    desc: Show available tasks
    cmds:
      - task --list

  # Development
  build:
    desc: Build the CLI binary
    cmds:
      - go build -o {{.BUILD_DIR}}/{{.BINARY_NAME}} ./cmd/alpaca

  run:
    desc: Run the CLI (pass args after --)
    cmds:
      - go run ./cmd/alpaca {{.CLI_ARGS}}

  # Testing
  test:
    desc: Run tests
    cmds:
      - go test -race ./...

  test:watch:
    desc: Run tests in watch mode
    cmds:
      - go install github.com/mitranim/gow@latest
      - gow test ./...

  # Code Quality
  lint:
    desc: Run linter
    cmds:
      - golangci-lint run ./...

  fmt:
    desc: Format code
    cmds:
      - goimports -w .

  check:
    desc: Run all checks (fmt, lint, test)
    cmds:
      - task: fmt
      - task: lint
      - task: test

  # Cleanup
  clean:
    desc: Clean build artifacts
    cmds:
      - rm -rf {{.BUILD_DIR}}
      - rm -f coverage.out

  # Dependencies
  deps:
    desc: Download dependencies
    cmds:
      - go mod download

  deps:tidy:
    desc: Tidy dependencies
    cmds:
      - go mod tidy

  # Tools
  tools:
    desc: Install development tools
    cmds:
      - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
      - go install golang.org/x/tools/cmd/goimports@latest
      - go install github.com/matryer/moq@latest

  # GUI (macOS)
  gui:build:
    desc: Build the GUI app
    dir: gui/Alpaca
    cmds:
      - xcodebuild -scheme Alpaca -configuration Release build

  gui:open:
    desc: Open GUI project in Xcode
    cmds:
      - open gui/Alpaca/Alpaca.xcodeproj
```

## .golangci.yml

```yaml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    # Default linters
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused

    # Additional linters
    - goimports       # Check import order
    - misspell        # Spell checking
    - errorlint       # Error wrapping best practices
    - nilerr          # nil error checks
    - exhaustive      # Switch exhaustiveness

linters-settings:
  goimports:
    local-prefixes: github.com/d2verb/alpaca

  errcheck:
    check-type-assertions: true
    check-blank: true

  govet:
    enable-all: true

  misspell:
    locale: US

  exhaustive:
    default-signifies-exhaustive: true

issues:
  exclude-rules:
    # Ignore test files for some linters
    - path: _test\.go
      linters:
        - errcheck
        - govet

  max-issues-per-linter: 0
  max-same-issues: 0
```

## .lefthook.yml

```yaml
pre-commit:
  parallel: true
  commands:
    goimports:
      glob: "*.go"
      run: goimports -w {staged_files}
      stage_fixed: true

    golangci-lint:
      glob: "*.go"
      run: golangci-lint run --fix {staged_files}
      stage_fixed: true

    swift-format:
      glob: "*.swift"
      run: swift-format -i {staged_files}
      stage_fixed: true

commit-msg:
  commands:
    gitmoji-check:
      run: |
        # Check if commit message starts with a gitmoji
        MSG=$(cat {1})
        # Common gitmoji patterns (emoji or :code: format)
        if ! echo "$MSG" | head -1 | grep -qE "^([\x{1F300}-\x{1F9FF}]|:[a-z_]+:)"; then
          echo "Error: Commit message must start with a Gitmoji"
          echo "Examples:"
          echo "  ✨ Add new feature"
          echo "  🐛 Fix bug"
          echo "  📝 Update documentation"
          echo ""
          echo "See https://gitmoji.dev for more"
          exit 1
        fi
```

## .github/workflows/ci.yml

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run tests
        run: go test -race ./...

  build:
    name: Build
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Build CLI
        run: go build -o alpaca ./cmd/alpaca

      - name: Build GUI
        run: |
          cd gui/Alpaca
          xcodebuild -scheme Alpaca -configuration Release build
```

## .github/workflows/release.yml

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release-cli:
    name: Release CLI
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

  release-gui:
    name: Release GUI
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build GUI
        run: |
          cd gui/Alpaca
          xcodebuild -scheme Alpaca -configuration Release archive \
            -archivePath build/Alpaca.xcarchive

      - name: Export app
        run: |
          cd gui/Alpaca
          xcodebuild -exportArchive \
            -archivePath build/Alpaca.xcarchive \
            -exportPath build/export \
            -exportOptionsPlist ExportOptions.plist

      - name: Create DMG
        run: |
          brew install create-dmg
          create-dmg \
            --volname "Alpaca" \
            --window-size 400 300 \
            --icon "Alpaca.app" 100 150 \
            --app-drop-link 300 150 \
            "Alpaca.dmg" \
            "gui/Alpaca/build/export/"

      - name: Upload to release
        uses: softprops/action-gh-release@v2
        with:
          files: Alpaca.dmg
```

## .goreleaser.yml

```yaml
version: 2

project_name: alpaca

before:
  hooks:
    - go mod tidy

builds:
  - id: alpaca
    main: ./cmd/alpaca
    binary: alpaca
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}

archives:
  - id: alpaca
    builds:
      - alpaca
    format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'

brews:
  - name: alpaca
    repository:
      owner: d2verb
      name: homebrew-tap
    homepage: https://github.com/d2verb/alpaca
    description: Lightweight llama-server wrapper for macOS
    license: MIT
    install: |
      bin.install "alpaca"
```
