#!/usr/bin/env bash
set -e

# ==============================================================================
# TIDE Installer
# Installs the latest pre-compiled binary for Linux / macOS / Windows
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/olifink/tide-go/main/install.sh | bash
# ==============================================================================

REPO="olifink/tide-go"
BINARY_NAME="tide"

# Colors
BOLD="\033[1m"
GREEN="\033[32m"
BLUE="\033[34m"
YELLOW="\033[33m"
RED="\033[31m"
RESET="\033[0m"

log_info() {
    echo -e "${BLUE}==>${RESET} ${BOLD}$1${RESET}"
}

log_success() {
    echo -e "${GREEN}✓${RESET} ${BOLD}$1${RESET}"
}

log_warn() {
    echo -e "${YELLOW}Warning:${RESET} $1"
}

log_error() {
    echo -e "${RED}Error:${RESET} $1" >&2
}

# 1. Detect Operating System
OS="$(uname -s)"
case "$OS" in
    Linux*)     PLATFORM="linux" ;;
    Darwin*)    PLATFORM="darwin" ;;
    MINGW*|MSYS*|CYGWIN*) PLATFORM="windows" ;;
    *)
        log_error "Unsupported operating system: $OS"
        exit 1
        ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   ARCH_NAME="amd64" ;;
    arm64|aarch64) ARCH_NAME="arm64" ;;
    *)
        log_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# 3. Determine Installation Directory
if [ -n "$BINDIR" ]; then
    INSTALL_DIR="$BINDIR"
elif [ "$PLATFORM" = "windows" ]; then
    INSTALL_DIR="${LOCALAPPDATA:-$HOME/AppData/Local}/Programs/tide"
else
    INSTALL_DIR="${HOME}/.local/bin"
fi

log_info "Installing TIDE for ${PLATFORM}/${ARCH_NAME} into ${INSTALL_DIR}..."

# 4. Fetch Latest Release Tag from GitHub
LATEST_TAG=""
if command -v curl >/dev/null 2>&1; then
    LATEST_JSON="$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
elif command -v wget >/dev/null 2>&1; then
    LATEST_JSON="$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null || true)"
fi

if [ -n "$LATEST_JSON" ]; then
    LATEST_TAG="$(echo "$LATEST_JSON" | grep -m 1 '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)"
fi

if [ -z "$LATEST_TAG" ]; then
    LATEST_TAG="v0.1.0"
fi

# 5. Download and Extract Archive
TEMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'tide_install')"
trap 'rm -rf "$TEMP_DIR"' EXIT

ARCHIVE_EXT="tar.gz"
if [ "$PLATFORM" = "windows" ]; then
    ARCHIVE_EXT="zip"
fi

ARCHIVE_NAME="tide_${LATEST_TAG}_${PLATFORM}_${ARCH_NAME}.${ARCHIVE_EXT}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"

# Fallback for unreleased tags: direct latest download url
if [ -z "$LATEST_TAG" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/tide_${PLATFORM}_${ARCH_NAME}.${ARCHIVE_EXT}"
fi

log_info "Downloading ${DOWNLOAD_URL}..."

DOWNLOAD_SUCCESS=false
if command -v curl >/dev/null 2>&1; then
    if curl -fL "$DOWNLOAD_URL" -o "${TEMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null; then
        DOWNLOAD_SUCCESS=true
    fi
elif command -v wget >/dev/null 2>&1; then
    if wget -q "$DOWNLOAD_URL" -O "${TEMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null; then
        DOWNLOAD_SUCCESS=true
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = false ]; then
    # Try latest release asset download URL format as fallback
    FALLBACK_URL="https://github.com/${REPO}/releases/latest/download/${ARCHIVE_NAME}"
    if command -v curl >/dev/null 2>&1; then
        curl -fL "$FALLBACK_URL" -o "${TEMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null && DOWNLOAD_SUCCESS=true || true
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$FALLBACK_URL" -O "${TEMP_DIR}/${ARCHIVE_NAME}" 2>/dev/null && DOWNLOAD_SUCCESS=true || true
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = false ]; then
    log_error "Failed to download release archive from GitHub. Please check https://github.com/${REPO}/releases"
    exit 1
fi

log_info "Extracting..."
if [ "$ARCHIVE_EXT" = "tar.gz" ]; then
    tar -xzf "${TEMP_DIR}/${ARCHIVE_NAME}" -C "${TEMP_DIR}"
else
    unzip -q "${TEMP_DIR}/${ARCHIVE_NAME}" -d "${TEMP_DIR}"
fi

# 6. Install Binary
mkdir -p "$INSTALL_DIR"

EXECUTABLE_NAME="$BINARY_NAME"
if [ "$PLATFORM" = "windows" ]; then
    EXECUTABLE_NAME="${BINARY_NAME}.exe"
fi

if [ ! -f "${TEMP_DIR}/${EXECUTABLE_NAME}" ]; then
    # Check inside nested extracted directories if any
    FOUND_BIN="$(find "${TEMP_DIR}" -name "${EXECUTABLE_NAME}" -type f | head -n 1)"
    if [ -n "$FOUND_BIN" ]; then
        cp "$FOUND_BIN" "${INSTALL_DIR}/${EXECUTABLE_NAME}"
    else
        log_error "Could not find ${EXECUTABLE_NAME} inside extracted archive"
        exit 1
    fi
else
    cp "${TEMP_DIR}/${EXECUTABLE_NAME}" "${INSTALL_DIR}/${EXECUTABLE_NAME}"
fi

chmod +x "${INSTALL_DIR}/${EXECUTABLE_NAME}"

log_success "TIDE ${LATEST_TAG} installed to ${INSTALL_DIR}/${EXECUTABLE_NAME}"

# 7. Check PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*)
        echo ""
        echo -e "${GREEN}🌊 Ready to code! Run ${BOLD}tide${RESET}${GREEN} to get started.${RESET}"
        ;;
    *)
        echo ""
        log_warn "${INSTALL_DIR} is not in your PATH."
        echo ""
        echo "Add it to your shell configuration to run 'tide' from anywhere:"
        echo ""
        if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
            echo -e "  ${BOLD}echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc${RESET}"
        elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
            echo -e "  ${BOLD}echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc && source ~/.bashrc${RESET}"
        elif [ -f "$HOME/.config/fish/config.fish" ]; then
            echo -e "  ${BOLD}fish_add_path ~/.local/bin${RESET}"
        else
            echo -e "  ${BOLD}export PATH=\"${INSTALL_DIR}:\$PATH\"${RESET}"
        fi
        echo ""
        ;;
esac
