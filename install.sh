#!/usr/bin/env bash
set -e

# Colors for output
GREEN="\033[32m"
BLUE="\033[34m"
RED="\033[31m"
RESET="\033[0m"
BOLD="\033[1m"

echo "${BLUE}${BOLD}Installing govm - Go Version Manager${RESET}"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "${RED}Error: Go is required to build govm. Please install Go first.${RESET}"
    exit 1
fi

# Define installation directories
GOVM_DIR="${HOME}/.govm"
GOVM_VERSIONS_DIR="${GOVM_DIR}/versions/go"
GOVM_CACHE_DIR="${GOVM_DIR}/.cache"
GOVM_BIN_DIR="/usr/local/bin"

# Check if user has sudo access for system-wide installation
HAS_SUDO=0
if command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
    HAS_SUDO=1
else
    echo "${BLUE}No sudo access detected. Installing govm locally.${RESET}"
    GOVM_BIN_DIR="${HOME}/.local/bin"
    mkdir -p "${GOVM_BIN_DIR}"
fi

# Create govm directories
echo "${BLUE}Creating govm directories...${RESET}"
mkdir -p "${GOVM_VERSIONS_DIR}"
mkdir -p "${GOVM_CACHE_DIR}"

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

SHELL_PROFILE=$(detect_shell_profile)

# Get temporary directory for building
TMP_DIR=$(mktemp -d)
cd "${TMP_DIR}"

# Clone repository or download source
if command -v git &> /dev/null; then
    echo "${BLUE}Cloning govm repository...${RESET}"
    git clone --depth 1 https://github.com/emmadal/govm.git
    cd govm
else
    echo "${BLUE}Downloading govm source...${RESET}"
    TARBALL_URL="https://github.com/emmadal/govm/archive/main.tar.gz"
    curl -sL "${TARBALL_URL}" | tar -xz
    cd govm-main
fi

# Build govm
echo "${BLUE}Building govm...${RESET}"
go build -ldflags "-s -w" -o govm .

# Install govm binary
echo "${BLUE}Installing govm binary...${RESET}"
if [[ "${HAS_SUDO}" -eq 1 && "${GOVM_BIN_DIR}" == "/usr/local/bin" ]]; then
    sudo cp govm "${GOVM_BIN_DIR}/"
    sudo chmod +x "${GOVM_BIN_DIR}/govm"
else
    cp govm "${GOVM_BIN_DIR}/"
    chmod +x "${GOVM_BIN_DIR}/govm"
fi

# Add bin directory to PATH if needed
if [[ "${GOVM_BIN_DIR}" == "${HOME}/.local/bin" ]]; then
    if ! grep -q "${GOVM_BIN_DIR}" "${SHELL_PROFILE}"; then
        echo "${BLUE}Adding ${GOVM_BIN_DIR} to your PATH in ${SHELL_PROFILE}${RESET}"
        echo "" >> "${SHELL_PROFILE}"
        echo "# govm installation" >> "${SHELL_PROFILE}"
        echo "export PATH=\"\${HOME}/.local/bin:\${PATH}\"" >> "${SHELL_PROFILE}"
    fi
fi

# Clean up temporary directory
cd
rm -rf "${TMP_DIR}"

echo "${GREEN}${BOLD}✓ govm has been successfully installed!${RESET}"
echo ""
echo "To start using govm, you may need to restart your terminal or run:"
echo "${BLUE}    source ${SHELL_PROFILE}${RESET}"
echo ""
echo "Usage examples:"
echo "${BLUE}    govm install 1.21.6  # Install Go 1.21.6${RESET}"
echo "${BLUE}    govm use 1.21.6      # Switch to Go 1.21.6${RESET}"
echo "${BLUE}    govm list            # List installed Go versions${RESET}"
echo "${BLUE}    govm rm 1.21.6       # Remove Go 1.21.6${RESET}"
echo ""
echo "For more information, visit: ${BLUE}https://github.com/emmadal/govm${RESET}"
