#!/usr/bin/env bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${BLUE}${BOLD}Installing GOVM - Go Version Manager${NC}"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Map architecture names to Go standard
case "${ARCH}" in
    x86_64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    i386|i686) ARCH="386" ;;
    *)
        echo -e "${RED}Unsupported architecture: ${ARCH}${NC}"
        echo "Please submit an issue at: https://github.com/emmadal/govm/issues"
        exit 1
        ;;
esac

# Function to detect shell and profile file
detect_shell_profile() {
    SHELL_NAME=$(basename "${SHELL}")
    
    case "${SHELL_NAME}" in
        bash)
            if [[ -f "${HOME}/.bash_profile" ]]; then
                echo "${HOME}/.bash_profile"
            else
                echo "${HOME}/.bashrc"
            fi
            ;;
        zsh)
            echo "${HOME}/.zshrc"
            ;;
        *)
            echo "${HOME}/.profile"
            ;;
    esac
}

# Detect shell profile
SHELL_PROFILE=$(detect_shell_profile)

# Setup directories
GOVM_DIR="${HOME}/.govm"
GOVM_VERSIONS_DIR="${GOVM_DIR}/versions/go"
GOVM_CACHE_DIR="${GOVM_DIR}/.cache"

# Determine bin directory based on sudo access
if command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
    BIN_DIR="/usr/local/bin"
else
    BIN_DIR="${HOME}/.local/bin"
    mkdir -p "${BIN_DIR}"
fi

# Create required directories
mkdir -p "${GOVM_VERSIONS_DIR}" "${GOVM_CACHE_DIR}"

# Create temp directory for download
TMP_DIR=$(mktemp -d)
cd "${TMP_DIR}"

# Download binary
DOWNLOAD_URL="https://github.com/emmadal/govm/releases/latest/download/govm_${OS}_${ARCH}"
echo -e "${BLUE}Downloading GOVM binary for ${OS}_${ARCH}...${NC}"

if command -v curl &> /dev/null; then
    curl -s -L -o govm "${DOWNLOAD_URL}"
elif command -v wget &> /dev/null; then
    wget -q -O govm "${DOWNLOAD_URL}"
else
    echo -e "${RED}Error: Neither curl nor wget found. Please install one of them and try again.${NC}"
    exit 1
fi

# Verify download success
if [ ! -s govm ]; then
    echo -e "${RED}Failed to download GOVM binary.${NC}"
    exit 1
fi

chmod +x govm

# Install binary
echo -e "${BLUE}Installing GOVM binary...${NC}"
if [[ "${BIN_DIR}" == "/usr/local/bin" ]]; then
    sudo cp govm "${BIN_DIR}/"
    sudo chmod +x "${BIN_DIR}/govm"
else
    cp govm "${BIN_DIR}/"
    chmod +x "${BIN_DIR}/govm"
fi

# Add bin directory to PATH if needed
if [[ "${BIN_DIR}" == "${HOME}/.local/bin" ]]; then
    if ! grep -q "${BIN_DIR}" "${SHELL_PROFILE}"; then
        echo "Adding ${BIN_DIR} to your PATH in ${SHELL_PROFILE}"
        echo "" >> "${SHELL_PROFILE}"
        echo "# govm installation" >> "${SHELL_PROFILE}"
        echo "export PATH=\"\${HOME}/.local/bin:\${PATH}\"" >> "${SHELL_PROFILE}"
    fi
fi

# Store version information by getting latest tag from GitHub
GOVM_VERSION=$(curl -s https://api.github.com/repos/emmadal/govm/releases/latest | grep tag_name | cut -d '"' -f 4)
echo "${GOVM_VERSION}" > "${BIN_DIR}/VERSION"
echo "$(date)" >> "${BIN_DIR}/VERSION"

# Clean up
cd
rm -rf "${TMP_DIR}"
echo ""

echo -e "${GREEN}${BOLD}🎉 GOVM has been successfully installed!${NC}"
echo ""
echo "To start using GOVM, you may need to restart your terminal or run:"
[[ -n "${SHELL_PROFILE}" ]] && echo -e "${BLUE}    source ${SHELL_PROFILE}${NC}"