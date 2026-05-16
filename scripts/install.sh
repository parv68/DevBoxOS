#!/usr/bin/env bash
set -euo pipefail

# DevBoxOS Installer
# Usage: curl -fsSL https://devbox.sh/install.sh | sh

REPO="devboxos/devboxos"
INSTALL_DIR="${DEVBOX_INSTALL_DIR:-/usr/local/bin}"
VERSION="${DEVBOX_VERSION:-latest}"
GITHUB_URL="https://github.com/${REPO}/releases"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}ℹ $1${NC}"; }
warn() { echo -e "${YELLOW}⚠ $1${NC}"; }
error() { echo -e "${RED}✗ $1${NC}" >&2; }

# Detect OS and Arch
detect_os() {
    case "$(uname -s)" in
        Darwin*)  echo "darwin" ;;
        Linux*)   echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)        echo "unknown" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             echo "amd64" ;;
    esac
}

# Check prerequisites
check_prereqs() {
    if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
        error "curl or wget is required"
        exit 1
    fi
}

# Get latest version from GitHub
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    fi
    info "Installing DevBoxOS ${VERSION}"
}

# Download binary
download_binary() {
    local os=$1
    local arch=$2
    local binary_name="devbox"
    local ext=""

    if [ "$os" = "windows" ]; then
        binary_name="devbox.exe"
        ext=".exe"
    fi

    local filename="devbox-${VERSION}-${os}-${arch}${ext}"
    local download_url="${GITHUB_URL}/download/${VERSION}/${filename}"

    info "Downloading ${filename}..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "/tmp/${filename}" "$download_url" || {
            error "Failed to download ${filename}"
            error "Check available releases at ${GITHUB_URL}"
            exit 1
        }
    else
        wget -q -O "/tmp/${filename}" "$download_url" || {
            error "Failed to download ${filename}"
            exit 1
        }
    fi

    # Verify download
    if [ ! -s "/tmp/${filename}" ]; then
        error "Downloaded file is empty"
        exit 1
    fi

    # Make executable
    chmod +x "/tmp/${filename}"

    # Install
    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"
    mv "/tmp/${filename}" "${INSTALL_DIR}/devbox${ext}"

    # Download engine
    local engine_filename="devbox-engine-${VERSION}-${os}-${arch}${ext}"
    local engine_url="${GITHUB_URL}/download/${VERSION}/${engine_filename}"

    info "Downloading engine..."
    if command -v curl &> /dev/null; then
        curl -fsSL -o "/tmp/${engine_filename}" "$engine_url" || {
            warn "Failed to download engine (optional)"
        }
    else
        wget -q -O "/tmp/${engine_filename}" "$engine_url" || {
            warn "Failed to download engine (optional)"
        }
    fi

    if [ -s "/tmp/${engine_filename}" ]; then
        chmod +x "/tmp/${engine_filename}"
        mv "/tmp/${engine_filename}" "${INSTALL_DIR}/devbox-engine${ext}"
    fi

    # Cleanup
    rm -f "/tmp/${filename}" "/tmp/${engine_filename}"
}

# Verify installation
verify_install() {
    if command -v devbox &> /dev/null; then
        info "DevBoxOS installed successfully!"
        echo ""
        echo "  Version: $(devbox version 2>/dev/null || echo 'unknown')"
        echo "  Location: $(which devbox)"
        echo ""
        echo "Get started:"
        echo "  devbox init          # Initialize a new project"
        echo "  devbox start         # Start your environment"
        echo "  devbox doctor        # Run diagnostics"
        echo ""
    else
        error "Installation failed - devbox not found in PATH"
        exit 1
    fi
}

# Main
main() {
    check_prereqs

    local os=$(detect_os)
    local arch=$(detect_arch)

    if [ "$os" = "unknown" ]; then
        error "Unsupported operating system"
        exit 1
    fi

    info "Detected: ${os}/${arch}"

    get_latest_version
    download_binary "$os" "$arch"
    verify_install
}

main "$@"
