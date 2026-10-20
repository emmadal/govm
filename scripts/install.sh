#!/usr/bin/env bash
# govm installer for Linux and macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/emmadal/govm/main/scripts/install.sh | bash
#
# Environment:
#   GOVM_BIN_DIR  where to put the govm binary (default: ~/.local/bin)
#   GOVM_DIR      where govm keeps Go versions (default: ~/.govm)
set -euo pipefail

GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

info() { echo -e "${BLUE}$*${NC}"; }
fail() { echo -e "${RED}$*${NC}" >&2; exit 1; }

info "${BOLD}Installing govm - Go Version Manager"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
    linux|darwin) ;;
    *) fail "Unsupported OS: ${OS}. On Windows, use scripts/install.ps1." ;;
esac

ARCH="$(uname -m)"
case "${ARCH}" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    i386|i686)     ARCH="386" ;;
    *) fail "Unsupported architecture: ${ARCH}. Please open an issue at https://github.com/emmadal/govm/issues" ;;
esac

BIN_DIR="${GOVM_BIN_DIR:-${HOME}/.local/bin}"
GOVM_ROOT="${GOVM_DIR:-${HOME}/.govm}"
mkdir -p "${BIN_DIR}" "${GOVM_ROOT}/versions/go" "${GOVM_ROOT}/.cache"

download() { # url dest
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$2" "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$2" "$1"
    else
        fail "Neither curl nor wget found. Please install one of them and try again."
    fi
}

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d ' ' -f 1
    else
        shasum -a 256 "$1" | cut -d ' ' -f 1
    fi
}

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

ASSET="govm_${OS}_${ARCH}"
BASE_URL="https://github.com/emmadal/govm/releases/latest/download"

info "Downloading ${ASSET}..."
download "${BASE_URL}/${ASSET}" "${TMP_DIR}/govm" || fail "Failed to download ${BASE_URL}/${ASSET}"
[ -s "${TMP_DIR}/govm" ] || fail "Downloaded file is empty."

if download "${BASE_URL}/checksums.txt" "${TMP_DIR}/checksums.txt" 2>/dev/null; then
    EXPECTED="$(awk -v a="${ASSET}" '{ sub(/^\*/, "", $2) } $2 == a { print $1 }' "${TMP_DIR}/checksums.txt")"
    [ -n "${EXPECTED}" ] || fail "checksums.txt has no entry for ${ASSET}."
    [ "$(sha256 "${TMP_DIR}/govm")" = "${EXPECTED}" ] || fail "Checksum verification failed for ${ASSET}."
    info "Checksum verified."
else
    echo "Warning: this release publishes no checksums; skipping verification." >&2
fi

chmod +x "${TMP_DIR}/govm"
mv "${TMP_DIR}/govm" "${BIN_DIR}/govm"

# Shell profile, matching the detection in pkg/shell.go.
case "$(basename "${SHELL:-sh}")" in
    zsh)  PROFILE="${HOME}/.zshrc" ;;
    bash) if [ "${OS}" = "darwin" ]; then PROFILE="${HOME}/.bash_profile"; else PROFILE="${HOME}/.bashrc"; fi ;;
    fish) PROFILE="${HOME}/.config/fish/conf.d/govm.fish" ;;
    *)    PROFILE="${HOME}/.profile" ;;
esac

# Paths under $HOME are written relative to it, so the profile stays portable.
home_relative() { case "$1" in "${HOME}"/*) echo "\$HOME/${1#"${HOME}"/}" ;; *) echo "$1" ;; esac; }
BIN_ENTRY="$(home_relative "${BIN_DIR}")"
CURRENT_ENTRY="$(home_relative "${GOVM_ROOT}/current/bin")"

# govm owns everything between these markers; `govm uninstall` removes it.
if ! grep -qs '^# >>> govm >>>' "${PROFILE}"; then
    mkdir -p "$(dirname "${PROFILE}")"
    {
        echo ""
        echo "# >>> govm >>>"
        if [ "${PROFILE##*.}" = "fish" ]; then
            [ -n "${GOVM_DIR:-}" ] && echo "set -gx GOVM_DIR \"$(home_relative "${GOVM_ROOT}")\""
            echo "fish_add_path --global --move --path \"${CURRENT_ENTRY}\" \"${BIN_ENTRY}\""
        else
            [ -n "${GOVM_DIR:-}" ] && echo "export GOVM_DIR=\"$(home_relative "${GOVM_ROOT}")\""
            echo "export PATH=\"${BIN_ENTRY}:${CURRENT_ENTRY}:\$PATH\""
        fi
        echo "# <<< govm <<<"
    } >> "${PROFILE}"
    info "Updated ${PROFILE}"
fi

echo ""
echo -e "${GREEN}${BOLD}🎉 govm has been successfully installed to ${BIN_DIR}/govm${NC}"
echo ""
echo "Open a new terminal, or run:"
echo -e "${BLUE}    source ${PROFILE}${NC}"
echo "then install Go with:"
echo -e "${BLUE}    govm install latest${NC}"
