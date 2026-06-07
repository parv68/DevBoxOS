#!/usr/bin/env bash
set -euo pipefail

# DevBoxOS Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/parv68/DevBoxOS/main/scripts/install.sh | sh

REPO="parv68/DevBoxOS"
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
    if ! command -v tar &> /dev/null && ! command -v unzip &> /dev/null; then
        error "tar (or unzip on Windows) is required"
        exit 1
    fi
}

# Get latest stable version from GitHub (by semver, not by publish date)
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases?per_page=30" | \
            grep '"tag_name"' | \
            sed -E 's/.*"([^"]+)".*/\1/' | \
            grep -v -- '-rc\|-alpha\|-beta' | \
            sed 's/^v//; s/-.*//' | \
            sort -t. -k1,1n -k2,2n -k3,3n | \
            tail -1 | \
            sed 's/^/v/')
    fi
    info "Installing DevBoxOS ${VERSION}"
}

# Download and extract archive
download_and_extract() {
    local os=$1
    local arch=$2
    local ext="tar.gz"
    local extract_cmd="tar xzf"

    if [ "$os" = "windows" ]; then
        ext="zip"
        extract_cmd="unzip -o"
    fi

    local archive_name="devbox_${VERSION}_${os}_${arch}.${ext}"
    local archive_url="${GITHUB_URL}/download/${VERSION}/${archive_name}"
    local tmp_dir
    tmp_dir=$(mktemp -d)

    info "Downloading ${archive_name}..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "${tmp_dir}/${archive_name}" "$archive_url" || {
            error "Failed to download ${archive_name}"
            error "Check available releases at ${GITHUB_URL}"
            exit 1
        }
    else
        wget -q -O "${tmp_dir}/${archive_name}" "$archive_url" || {
            error "Failed to download ${archive_name}"
            exit 1
        }
    fi

    # Verify download
    if [ ! -s "${tmp_dir}/${archive_name}" ]; then
        error "Downloaded file is empty"
        exit 1
    fi

    info "Extracting..."
    (cd "$tmp_dir" && $extract_cmd "${archive_name}")

    # Install binaries
    info "Installing to ${INSTALL_DIR}..."
    mkdir -p "$INSTALL_DIR"

    if [ "$os" = "windows" ]; then
        install -m 755 "${tmp_dir}/devbox.exe" "${INSTALL_DIR}/devbox.exe" 2>/dev/null || \
            cp "${tmp_dir}/devbox.exe" "${INSTALL_DIR}/devbox.exe"
        if [ -f "${tmp_dir}/devbox-engine.exe" ]; then
            cp "${tmp_dir}/devbox-engine.exe" "${INSTALL_DIR}/devbox-engine.exe" 2>/dev/null || true
        fi
    else
        install -m 755 "${tmp_dir}/devbox" "${INSTALL_DIR}/devbox" 2>/dev/null || \
            cp "${tmp_dir}/devbox" "${INSTALL_DIR}/devbox"
        if [ -f "${tmp_dir}/devbox-engine" ]; then
            install -m 755 "${tmp_dir}/devbox-engine" "${INSTALL_DIR}/devbox-engine" 2>/dev/null || \
                cp "${tmp_dir}/devbox-engine" "${INSTALL_DIR}/devbox-engine"
        fi
    fi

    # Cleanup
    rm -rf "$tmp_dir"
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
        echo "  devbox init            # Initialize a new project"
        echo "  cd my-project"
        echo "  devbox start           # Start your environment"
        echo "  devbox doctor          # Run diagnostics"
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
    download_and_extract "$os" "$arch"
    verify_install
}

main "$@"
