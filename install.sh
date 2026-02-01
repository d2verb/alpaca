#!/bin/sh
set -e

# Alpaca installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/d2verb/alpaca/main/install.sh | sh

REPO="d2verb/alpaca"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       echo "unsupported" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             echo "unsupported" ;;
    esac
}

# Get latest version from GitHub API
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Get receipt directory (handles sudo case)
get_receipt_dir() {
    if [ -n "${SUDO_USER:-}" ]; then
        # Running via sudo - use the original user's home
        if command -v getent >/dev/null 2>&1; then
            # Linux: use getent to get home directory
            echo "$(getent passwd "$SUDO_USER" | cut -d: -f6)/.alpaca"
        else
            # macOS: getent not available, use /Users
            echo "/Users/$SUDO_USER/.alpaca"
        fi
    else
        echo "$HOME/.alpaca"
    fi
}

# Portable SHA256 computation (works on macOS and Linux)
compute_sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | cut -d' ' -f1
    else
        echo "Error: No SHA256 tool found (sha256sum or shasum required)" >&2
        exit 1
    fi
}

# Create installation receipt for upgrade command
create_receipt() {
    RECEIPT_DIR=$(get_receipt_dir)
    BINARY_PATH="${INSTALL_DIR}/alpaca"

    mkdir -p "$RECEIPT_DIR"

    FINGERPRINT=$(compute_sha256 "$BINARY_PATH")
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    cat > "$RECEIPT_DIR/install.json" << EOF
{
  "schema_version": 1,
  "install_source": "script",
  "installed_at": "$TIMESTAMP",
  "binary_path": "$BINARY_PATH",
  "binary_fingerprint": "sha256:$FINGERPRINT"
}
EOF

    # If running via sudo, fix ownership of receipt directory
    if [ -n "${SUDO_USER:-}" ]; then
        chown -R "$SUDO_USER" "$RECEIPT_DIR"
    fi

    echo "Created installation receipt at $RECEIPT_DIR/install.json"
}

main() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    if [ "$OS" = "unsupported" ]; then
        echo "Error: Unsupported operating system: $(uname -s)"
        exit 1
    fi

    if [ "$ARCH" = "unsupported" ]; then
        echo "Error: Unsupported architecture: $(uname -m)"
        exit 1
    fi

    echo "Detected: ${OS}/${ARCH}"

    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version"
        exit 1
    fi

    # Remove 'v' prefix for filename
    VERSION_NUM="${VERSION#v}"
    FILENAME="alpaca_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

    echo "Downloading alpaca ${VERSION}..."

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    curl -fsSL "$URL" -o "${TMPDIR}/${FILENAME}"
    tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"

    echo "Installing to ${INSTALL_DIR}/alpaca..."

    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMPDIR}/alpaca" "${INSTALL_DIR}/alpaca"
    else
        sudo mv "${TMPDIR}/alpaca" "${INSTALL_DIR}/alpaca"
    fi

    # Create receipt for upgrade command
    create_receipt

    echo "Done! Run 'alpaca --help' to get started."
}

main
