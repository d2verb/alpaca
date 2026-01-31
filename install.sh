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

    echo "Done! Run 'alpaca --help' to get started."
}

main
