#!/usr/bin/env bash
set -e

# Colors for output
GREEN=$(printf '\033[32m')
BLUE=$(printf '\033[34m')
RED=$(printf '\033[31m')
RESET=$(printf '\033[0m')
BOLD=$(printf '\033[1m')

echo "${RED}${BOLD}Removing govm - Go Version Manager${RESET}"

# Define installation directories
GOVM_DIR="${HOME}/.govm"
GOVM_BIN_DIR="/usr/local/bin"

# Check if user has sudo access
HAS_SUDO=0
if command -v sudo &> /dev/null && sudo -n true 2>/dev/null; then
    HAS_SUDO=1
else
    echo "${BLUE}No sudo access detected. Assuming govm was installed locally.${RESET}"
    GOVM_BIN_DIR="${HOME}/.local/bin"
fi

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

# Remove govm binary
echo "${BLUE}Removing govm binary...${RESET}"
if [[ -f "${GOVM_BIN_DIR}/govm" ]]; then
    if [[ "${HAS_SUDO}" -eq 1 && "${GOVM_BIN_DIR}" == "/usr/local/bin" ]]; then
        sudo rm -f "${GOVM_BIN_DIR}/govm"
    else
        rm -f "${GOVM_BIN_DIR}/govm"
    fi
    echo "${GREEN}✓ Removed govm binary${RESET}"
else
    echo "${RED}govm binary not found in ${GOVM_BIN_DIR}${RESET}"
fi

# Remove govm directories
echo "${BLUE}Removing govm directories...${RESET}"
if [[ -d "${GOVM_DIR}" ]]; then
    rm -rf "${GOVM_DIR}"
    echo "${GREEN}✓ Removed govm directories${RESET}"
else
    echo "${RED}govm directory not found at ${GOVM_DIR}${RESET}"
fi

# Clean shell profile
echo "${BLUE}Updating shell profile (${SHELL_PROFILE})...${RESET}"
if [[ -f "${SHELL_PROFILE}" ]]; then
    # Create a temporary file
    TEMP_PROFILE=$(mktemp)
    
    # Filter out govm-related lines
    grep -v "# govm installation" "${SHELL_PROFILE}" | \
    grep -v "export PATH=\"\${HOME}/.local/bin:\${PATH}\"" | \
    grep -v "export PATH=\"\${GOVM_DIR}/versions/go" | \
    grep -v "export GOROOT=" > "${TEMP_PROFILE}"
    
    # Replace original file with cleaned version
    mv "${TEMP_PROFILE}" "${SHELL_PROFILE}"
    echo "${GREEN}✓ Updated shell profile${RESET}"
else
    echo "${RED}Shell profile not found at ${SHELL_PROFILE}${RESET}"
fi

echo "${GREEN}${BOLD}✓ govm has been successfully removed from your system!${RESET}"
echo ""
echo "${BLUE}To ensure all changes take effect, please restart your terminal or run:${RESET}"
echo "    source ${SHELL_PROFILE}"
echo ""
echo "Thank you for using govm!"
