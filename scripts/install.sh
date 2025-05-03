#!/usr/bin/env bash
set -e

# Colors for output
GREEN=$(printf '\033[32m')
BLUE=$(printf '\033[34m')
RED=$(printf '\033[31m')
RESET=$(printf '\033[0m')
BOLD=$(printf '\033[1m')

echo -e "${BLUE}${BOLD}Installing govm - Go Version Manager${RESET}"

# Ensure we're on a Unix-based system (macOS or Linux)
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "$OS" != "darwin" && "$OS" != "linux" ]]; then
    echo -e "${RED}Error: This script only supports macOS and Linux.${RESET}"
    echo -e "${RED}For Windows, please use the PowerShell script instead.${RESET}"
    exit 1
fi

# Detect architecture
ARCH="$(uname -m)"

# Map architecture names to Go standard
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
        echo -e "${RED}Unsupported architecture: ${ARCH}${RESET}"
        echo "Please submit an issue at: https://github.com/emmadal/govm/issues"
        exit 1
        ;;
esac

# Define Unix installation directories
GOVM_DIR="${HOME}/.govm"
GOVM_VERSIONS_DIR="${GOVM_DIR}/versions/go"
GOVM_CACHE_DIR="${GOVM_DIR}/.cache"
GOVM_BIN_DIR="/usr/local/bin"

# Check if user has sudo access for system-wide installation
HAS_SUDO=0
if command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
    HAS_SUDO=1
else
    echo -e "${BLUE}No sudo access detected. Installing govm locally.${RESET}"
    GOVM_BIN_DIR="${HOME}/.local/bin"
    mkdir -p "${GOVM_BIN_DIR}"
fi

# Create govm directories
echo -e "${BLUE}Creating govm directories...${RESET}"
mkdir -p "${GOVM_VERSIONS_DIR}"
mkdir -p "${GOVM_CACHE_DIR}"

# Function to detect shell and profile file (Unix only)
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

SHELL_PROFILE=$(detect_shell_profile)

# Create a temporary directory
TMP_DIR=$(mktemp -d)
cd "${TMP_DIR}"

# Get the latest version tag
echo -e "${BLUE}Retrieving latest version information...${RESET}"
if command -v curl &> /dev/null; then
    LATEST_VERSION=$(curl -s https://api.github.com/repos/emmadal/govm/releases/latest | grep -o '"tag_name":"[^"]*' | grep -o '[^"]*$')
elif command -v wget &> /dev/null; then
    LATEST_VERSION=$(wget -q -O - https://api.github.com/repos/emmadal/govm/releases/latest | grep -o '"tag_name":"[^"]*' | grep -o '[^"]*$')
else
    LATEST_VERSION="unknown"
fi

# Download the pre-compiled binary for the detected platform
RELEASE_URL="https://github.com/emmadal/govm/releases/latest/download/govm_${OS}_${ARCH}"
DOWNLOAD_URL="${RELEASE_URL}"

echo -e "${BLUE}Downloading govm binary for ${OS}_${ARCH}...${RESET}"
if command -v curl &> /dev/null; then
    curl -s -L -o govm "${DOWNLOAD_URL}"
elif command -v wget &> /dev/null; then
    wget -q -O govm "${DOWNLOAD_URL}"
else
    echo -e "${RED}Error: Neither curl nor wget found. Please install one of them and try again.${RESET}"
    exit 1
fi

# Check if download was successful
if [ ! -s govm ]; then
    echo -e "${RED}Failed to download govm binary.${RESET}"
    echo -e "${BLUE}Trying to download source code instead...${RESET}"
    
    # Try to download source as a fallback
    if command -v git &> /dev/null; then
        git clone --depth 1 https://github.com/emmadal/govm.git
        cd govm
    else
        curl -s -L -o master.tar.gz https://github.com/emmadal/govm/archive/main.tar.gz
        tar -xzf master.tar.gz
        cd govm-main
    fi
    
    echo -e "${RED}To build govm from source, you need Go installed on your machine.${RESET}"
    echo -e "${BLUE}Please install Go and then run:${RESET}"
    echo "cd $(pwd) && go build -ldflags=\"-s -w\" -o govm && sudo mv govm ${GOVM_BIN_DIR}/"
    exit 1
fi

chmod +x govm

# Install govm binary (Unix-specific)
echo -e "${BLUE}Installing govm binary...${RESET}"
if [[ "${HAS_SUDO}" -eq 1 && "${GOVM_BIN_DIR}" == "/usr/local/bin" ]]; then
    sudo cp govm "${GOVM_BIN_DIR}/"
    sudo chmod +x "${GOVM_BIN_DIR}/govm"
else
    cp govm "${GOVM_BIN_DIR}/"
    chmod +x "${GOVM_BIN_DIR}/govm"
fi

# Add bin directory to PATH if needed (Unix shell specific)
if [[ "${GOVM_BIN_DIR}" == "${HOME}/.local/bin" ]]; then
    if ! grep -q "${GOVM_BIN_DIR}" "${SHELL_PROFILE}"; then
        echo -e "${BLUE}Adding ${GOVM_BIN_DIR} to your PATH in ${SHELL_PROFILE}${RESET}"
        # shellcheck disable=SC2129
        echo "" >> "${SHELL_PROFILE}"
        echo "# govm installation" >> "${SHELL_PROFILE}"
        echo "export PATH=\"\${HOME}/.local/bin:\${PATH}\"" >> "${SHELL_PROFILE}"
    fi
fi

# Create VERSION file with the latest version tag
echo "${LATEST_VERSION}" > "${HOME}/.local/bin/VERSION"

# Clean up temporary directory (Unix-specific)
cd
rm -rf "${TMP_DIR}"

echo -e "${GREEN}${BOLD}🎉 govm has been successfully installed!${RESET}"
echo ""
echo "To start using govm, you may need to restart your terminal or run:"
echo -e "${BLUE}    source ${SHELL_PROFILE}${RESET}"
