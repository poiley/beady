#!/usr/bin/env bash
#
# Install script for bdy (beady) - a k9s-style TUI for beads
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/poiley/beady/main/scripts/install.sh | bash
#
# Options (via env vars):
#   BDY_VERSION=v0.2.0  Install a specific version (default: latest)
#   BDY_INSTALL_DIR=/usr/local/bin  Install directory (default: /usr/local/bin or ~/.local/bin)
#
set -euo pipefail

REPO_OWNER="poiley"
REPO_NAME="beady"
BINARY_NAME="bdy"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}==> ${NC}$1"; }
success() { echo -e "${GREEN}==> ${NC}$1"; }
warn() { echo -e "${YELLOW}==> ${NC}$1"; }
error() { echo -e "${RED}==> ERROR: ${NC}$1" >&2; }

# Detect OS and architecture
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)  os="linux";;
        Darwin*) os="darwin";;
        MINGW*|MSYS*|CYGWIN*) os="windows";;
        *)
            error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64";;
        arm64|aarch64) arch="arm64";;
        *)
            error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac

    echo "${os}_${arch}"
}

# Get the latest release version from GitHub
get_latest_version() {
    local url="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
    local version

    if command -v curl &>/dev/null; then
        version=$(curl -fsSL "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    elif command -v wget &>/dev/null; then
        version=$(wget -qO- "$url" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
    else
        error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    if [ -z "$version" ]; then
        error "Could not determine latest version. Check https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
        exit 1
    fi

    echo "$version"
}

# Determine install directory
get_install_dir() {
    if [ -n "${BDY_INSTALL_DIR:-}" ]; then
        echo "$BDY_INSTALL_DIR"
        return
    fi

    # Prefer /usr/local/bin if writable, otherwise ~/.local/bin
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
    else
        local dir="${HOME}/.local/bin"
        mkdir -p "$dir"
        echo "$dir"
    fi
}

# Download and install
install() {
    local platform version install_dir archive_name download_url tmp_dir

    platform=$(detect_platform)
    version="${BDY_VERSION:-$(get_latest_version)}"
    install_dir=$(get_install_dir)

    # Strip v prefix for archive name
    local version_clean="${version#v}"
    archive_name="${BINARY_NAME}_${platform}.tar.gz"
    download_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${version}/${archive_name}"

    info "Installing ${BINARY_NAME} ${version} for ${platform}..."
    info "Download URL: ${download_url}"
    info "Install directory: ${install_dir}"

    # Create temp directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "${tmp_dir:-}"' EXIT

    # Download
    info "Downloading..."
    if command -v curl &>/dev/null; then
        curl -fsSL "$download_url" -o "${tmp_dir}/${archive_name}"
    else
        wget -q "$download_url" -O "${tmp_dir}/${archive_name}"
    fi

    # Extract
    info "Extracting..."
    tar xzf "${tmp_dir}/${archive_name}" -C "$tmp_dir"

    # Find binary (goreleaser may nest in a directory)
    local bin_path="${tmp_dir}/${BINARY_NAME}"
    if [ ! -f "$bin_path" ]; then
        bin_path=$(find "$tmp_dir" -name "$BINARY_NAME" -type f | head -1)
    fi

    if [ -z "$bin_path" ] || [ ! -f "$bin_path" ]; then
        error "Binary not found in archive."
        exit 1
    fi

    # Install
    chmod +x "$bin_path"
    mv "$bin_path" "${install_dir}/${BINARY_NAME}"

    # Verify
    if [ -x "${install_dir}/${BINARY_NAME}" ]; then
        success "Installed ${BINARY_NAME} ${version} to ${install_dir}/${BINARY_NAME}"
        "${install_dir}/${BINARY_NAME}" --version
    else
        success "Installed to ${install_dir}/${BINARY_NAME}"
        warn "Make sure ${install_dir} is in your PATH."
        echo ""
        echo "  Add to your shell profile:"
        echo "    export PATH=\"${install_dir}:\$PATH\""
    fi

    echo ""
    info "Quick start:"
    echo "  cd your-project"
    echo "  bd init           # Initialize beads (if not already done)"
    echo "  bdy               # Launch the TUI"
}

# Check for existing installation
check_existing() {
    if command -v "$BINARY_NAME" &>/dev/null; then
        local current_version
        current_version=$("$BINARY_NAME" --version 2>/dev/null || echo "unknown")
        warn "Existing installation found: ${current_version}"
        warn "It will be replaced."
    fi
}

# Check if go install fallback is possible
try_go_install() {
    if command -v go &>/dev/null; then
        warn "Falling back to 'go install'..."
        local version="${BDY_VERSION:-latest}"
        go install -ldflags "-s -w -X 'main.Version=${version}'" "github.com/${REPO_OWNER}/${REPO_NAME}/cmd/bdy@${version}"
        success "Installed via go install."
        return 0
    fi
    return 1
}

main() {
    echo ""
    echo "  bdy installer - a k9s-style TUI for beads"
    echo ""

    check_existing

    # Try binary download first, fall back to go install
    if ! install 2>/dev/null; then
        warn "Binary download failed."
        if ! try_go_install; then
            error "Installation failed. Please install manually:"
            echo "  go install github.com/${REPO_OWNER}/${REPO_NAME}/cmd/bdy@latest"
            echo "  # or download from: https://github.com/${REPO_OWNER}/${REPO_NAME}/releases"
            exit 1
        fi
    fi
}

main "$@"
