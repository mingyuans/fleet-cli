#!/bin/sh
set -e

REPO="mingyuans/fleet-cli"
BINARY="fleet"
INSTALL_DIR="/usr/local/bin"

# Allow overrides via environment
VERSION="${FLEET_VERSION:-latest}"
INSTALL_DIR="${FLEET_INSTALL_DIR:-$INSTALL_DIR}"

log()  { printf '  \033[1m%s\033[0m\n' "$*"; }
ok()   { printf '  \033[32m✓\033[0m %s\n' "$*"; }
fail() { printf '  \033[31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# --- Detect OS ---
detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "darwin" ;;
        Linux*)  echo "linux" ;;
        *)       fail "Unsupported OS: $(uname -s). Only macOS and Linux are supported." ;;
    esac
}

# --- Detect architecture ---
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        arm64|aarch64)  echo "arm64" ;;
        *)              fail "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported." ;;
    esac
}

# --- Resolve version tag ---
resolve_version() {
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' | head -1 | cut -d'"' -f4)
        if [ -z "$VERSION" ]; then
            fail "Could not determine latest version. Set FLEET_VERSION explicitly."
        fi
    fi
}

# --- Verify checksum ---
verify_checksum() {
    checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"
    if command -v sha256sum >/dev/null 2>&1; then
        sha_cmd="sha256sum"
    elif command -v shasum >/dev/null 2>&1; then
        sha_cmd="shasum -a 256"
    else
        log "Skipping checksum verification (no sha256sum or shasum found)"
        return 0
    fi

    expected=$(curl -sSfL "$checksums_url" | grep "$1" | awk '{print $1}')
    if [ -z "$expected" ]; then
        log "Skipping checksum verification (asset not found in checksums.txt)"
        return 0
    fi

    actual=$($sha_cmd "$2" | awk '{print $1}')
    if [ "$expected" != "$actual" ]; then
        fail "Checksum mismatch for $1\n  expected: $expected\n  actual:   $actual"
    fi
    ok "Checksum verified"
}

# --- Main ---
main() {
    log "Installing ${BINARY}..."

    OS=$(detect_os)
    ARCH=$(detect_arch)
    resolve_version

    ASSET="${BINARY}-${OS}-${ARCH}.tar.gz"
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

    log "Version:  ${VERSION}"
    log "Platform: ${OS}/${ARCH}"
    log "URL:      ${URL}"

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    log "Downloading..."
    curl -sSfL -o "${TMPDIR}/${ASSET}" "$URL" \
        || fail "Download failed. Check that ${VERSION} exists at https://github.com/${REPO}/releases"

    ok "Downloaded ${ASSET}"

    verify_checksum "$ASSET" "${TMPDIR}/${ASSET}"

    log "Extracting..."
    tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"
    ok "Extracted"

    log "Installing to ${INSTALL_DIR}..."
    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
    fi
    if [ -w "$INSTALL_DIR" ]; then
        mv "${TMPDIR}/${BINARY}-${OS}-${ARCH}" "${INSTALL_DIR}/${BINARY}"
    else
        sudo mv "${TMPDIR}/${BINARY}-${OS}-${ARCH}" "${INSTALL_DIR}/${BINARY}"
    fi
    chmod +x "${INSTALL_DIR}/${BINARY}"
    ok "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"

    echo ""
    "${INSTALL_DIR}/${BINARY}" --version
    echo ""
    log "Run 'fleet --help' to get started."
}

main
