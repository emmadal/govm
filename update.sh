#!/usr/bin/env bash
set -e

# Colors for output
GREEN=$(printf '\033[32m')
BLUE=$(printf '\033[34m')
RED=$(printf '\033[31m')
RESET=$(printf '\033[0m')
BOLD=$(printf '\033[1m')

echo "${BLUE}${BOLD}Updating govm - Go Version Manager${RESET}"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Map architecture names
case "${ARCH}" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    i386|i686)
        ARCH="386"
        ;;
    *)
        echo "${RED}Unsupported architecture: ${ARCH}${RESET}"
        echo "Please submit an issue at: https://github.com/emmadal/govm/issues"
        exit 1
        ;;
esac

# Define installation directories
GOVM_BIN_DIR="/usr/local/bin"

# Check if user has sudo access for system-wide installation
HAS_SUDO=0
if command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
    HAS_SUDO=1
else
    echo "${BLUE}No sudo access detected. Updating govm locally.${RESET}"
    GOVM_BIN_DIR="${HOME}/.local/bin"
    mkdir -p "${GOVM_BIN_DIR}"
fi

# Create a temporary directory
TMP_DIR=$(mktemp -d)
cd "${TMP_DIR}"

# Download the pre-compiled binary for the detected platform
RELEASE_URL="https://github.com/emmadal/govm/releases/latest/download/govm_${OS}_${ARCH}"
DOWNLOAD_URL="${RELEASE_URL}"

echo "${BLUE}Downloading latest govm binary for ${OS}_${ARCH}...${RESET}"
if command -v curl &> /dev/null; then
    curl -s -L -o govm "${DOWNLOAD_URL}"
elif command -v wget &> /dev/null; then
    wget -q -O govm "${DOWNLOAD_URL}"
else
    echo "${RED}Error: Neither curl nor wget found. Please install one of them and try again.${RESET}"
    exit 1
fi

# Check if download was successful
if [ ! -s govm ]; then
    echo "${RED}Failed to download govm binary.${RESET}"
    echo "${RED}Update failed.${RESET}"
    exit 1
fi

chmod +x govm

# Install govm binary
echo "${BLUE}Updating govm binary...${RESET}"
if [[ "${HAS_SUDO}" -eq 1 && "${GOVM_BIN_DIR}" == "/usr/local/bin" ]]; then
    sudo cp govm "${GOVM_BIN_DIR}/"
    sudo chmod +x "${GOVM_BIN_DIR}/govm"
else
    cp govm "${GOVM_BIN_DIR}/"
    chmod +x "${GOVM_BIN_DIR}/govm"
fi

# Clean up temporary directory
cd
rm -rf "${TMP_DIR}"

echo "${GREEN}${BOLD}✓ govm has been successfully updated!${RESET}"
echo ""
echo "For more information, visit: ${BLUE}https://github.com/emmadal/govm${RESET}"
