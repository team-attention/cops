#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="$HOME/.cops/bin"
CDN_BASE_URL="https://storage.googleapis.com/cops-client/releases"
PROJECT_NAME="cops"

# Print colored message (to stderr to avoid capturing in subshells)
info() {
    echo -e "${GREEN}==>${NC} $1" >&2
}

warn() {
    echo -e "${YELLOW}Warning:${NC} $1" >&2
}

error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            echo "darwin"
            ;;
        Linux*)
            echo "linux"
            ;;
        *)
            error "Unsupported OS: $(uname -s). Only macOS and Linux are supported."
            ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64)
            echo "amd64"
            ;;
        arm64|aarch64)
            echo "arm64"
            ;;
        *)
            error "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported."
            ;;
    esac
}

# Detect SHA256 checksum command.
# Sets the global variable SHA256_CMD (intentionally global) used by verify_checksum.
detect_sha256_cmd() {
    if command -v sha256sum >/dev/null 2>&1; then
        SHA256_CMD="sha256sum"
    elif command -v shasum >/dev/null 2>&1; then
        SHA256_CMD="shasum -a 256"
    else
        error "No SHA256 checksum tool found. Install sha256sum or shasum."
    fi
}

# Get latest version from CDN or use COPS_VERSION override
get_latest_version() {
    local version="${COPS_VERSION:-}"

    if [ -z "$version" ]; then
        info "Fetching latest version from CDN..."
        version=$(curl -fsSL "${CDN_BASE_URL}/latest")

        if [ -z "$version" ]; then
            error "Failed to fetch latest version from CDN"
        fi

        # Trim trailing whitespace/newline only (preserves internal spaces if any)
        version=$(echo "$version" | sed 's/[[:space:]]*$//')
    fi

    echo "$version"
}

# Check if this is an upgrade
is_upgrade() {
    [ -f "$INSTALL_DIR/cops" ] || [ -f "$INSTALL_DIR/cops-daemon" ]
}

# Fetch latest version from CDN unconditionally (used for fallback in download)
fetch_latest_version_from_cdn() {
    local ver
    ver=$(curl -fsSL "${CDN_BASE_URL}/latest")
    if [ -z "$ver" ]; then
        error "Failed to fetch latest version from CDN"
    fi
    ver=$(echo "$ver" | sed 's/[[:space:]]*$//')
    echo "$ver"
}

# Verify SHA256 checksum of downloaded archive.
# Parameters: tmp_dir (explicit), archive_name, version
verify_checksum() {
    local tmp_dir=$1
    local archive_name=$2
    local version=$3
    local archive_path="$tmp_dir/$archive_name"

    info "Downloading checksums..."
    local checksums_url="${CDN_BASE_URL}/${version}/checksums.txt"
    if ! curl -fsSL "$checksums_url" -o "$tmp_dir/checksums.txt"; then
        rm -rf "$tmp_dir"
        error "Failed to download checksums from $checksums_url"
    fi

    # Extract expected checksum for our archive.
    # checksums.txt uses GoReleaser format: "<hash>  <filename>" (two spaces, standard sha256sum output).
    local expected_hash
    expected_hash=$(grep "  ${archive_name}$" "$tmp_dir/checksums.txt" | awk '{print $1}')

    if [ -z "$expected_hash" ]; then
        rm -rf "$tmp_dir"
        error "Archive $archive_name not found in checksums.txt"
    fi

    # Compute actual checksum using platform-detected SHA256_CMD (set by detect_sha256_cmd)
    info "Verifying checksum..."
    local actual_hash
    actual_hash=$($SHA256_CMD "$archive_path" | awk '{print $1}')

    if [ "$expected_hash" != "$actual_hash" ]; then
        rm -rf "$tmp_dir"
        printf "${RED}Error:${NC} Checksum verification failed for %s\n" "$archive_name" >&2
        printf "  Expected: %s\n" "$expected_hash" >&2
        printf "  Actual:   %s\n" "$actual_hash" >&2
        exit 1
    fi

    info "Checksum verified successfully"
}

# Download, verify, and extract archive.
# Sets global INSTALLED_VERSION with the actual version used (may differ from input due to fallback).
download_and_extract() {
    local version=$1
    local os=$2
    local arch=$3

    local archive_name="${PROJECT_NAME}_${version#v}_${os}_${arch}.tar.gz"
    local download_url="${CDN_BASE_URL}/${version}/${archive_name}"
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Ensure temp directory is cleaned up on unexpected exit (SIGINT, SIGTERM)
    trap 'rm -rf "$tmp_dir"' EXIT

    info "Downloading $archive_name..."
    if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
        if [ -n "${COPS_VERSION:-}" ]; then
            warn "Version $version not found on CDN. Falling back to latest release..."
            rm -rf "$tmp_dir"
            version=$(fetch_latest_version_from_cdn)
            info "Using latest version: $version"
            archive_name="${PROJECT_NAME}_${version#v}_${os}_${arch}.tar.gz"
            download_url="${CDN_BASE_URL}/${version}/${archive_name}"
            tmp_dir=$(mktemp -d)
            # Re-set trap for new tmp_dir
            trap 'rm -rf "$tmp_dir"' EXIT
            if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
                rm -rf "$tmp_dir"
                error "Failed to download $download_url"
            fi
        else
            rm -rf "$tmp_dir"
            error "Failed to download $download_url"
        fi
    fi

    # Verify checksum before extraction (covers both direct download and fallback paths,
    # since $version and $archive_name are updated in the fallback branch above)
    verify_checksum "$tmp_dir" "$archive_name" "$version"

    info "Extracting archive..."
    tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"

    # Create install directory
    mkdir -p "$INSTALL_DIR"

    # Move binaries
    info "Installing binaries to $INSTALL_DIR..."
    mv "$tmp_dir/cops" "$INSTALL_DIR/cops"
    mv "$tmp_dir/cops-daemon" "$INSTALL_DIR/cops-daemon"

    # Make executable
    chmod +x "$INSTALL_DIR/cops"
    chmod +x "$INSTALL_DIR/cops-daemon"

    # Cleanup and clear trap
    rm -rf "$tmp_dir"
    trap - EXIT

    # Set global variable with actual version used (avoids stdout capture fragility)
    INSTALLED_VERSION="$version"
}

# Detect shell and config file
get_shell_config() {
    local shell_name=$(basename "$SHELL")

    case "$shell_name" in
        zsh)
            if [ -f "$HOME/.zprofile" ]; then
                echo "$HOME/.zprofile"
            else
                echo "$HOME/.zshrc"
            fi
            ;;
        bash)
            if [ -f "$HOME/.bash_profile" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.bashrc"
            fi
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

# Update PATH in shell config
update_path() {
    local config_file=$(get_shell_config)
    local path_export="export PATH=\"\$HOME/.cops/bin:\$PATH\""

    # Check if PATH is already configured
    if grep -q ".cops/bin" "$config_file" 2>/dev/null; then
        info "PATH already configured in $config_file"
        return
    fi

    info "Adding $INSTALL_DIR to PATH in $config_file..."
    echo "" >> "$config_file"
    echo "# Added by cops installer" >> "$config_file"
    echo "$path_export" >> "$config_file"

    warn "Please restart your shell or run: source $config_file"
}

# Register daemon service
register_daemon() {
    local upgrade=$1

    # Check if cops is in PATH (either from current session or will be after shell restart)
    local cops_bin="$INSTALL_DIR/cops"

    if [ "$upgrade" = true ]; then
        info "Upgrading: Uninstalling existing daemon service..."
        if "$cops_bin" uninstall 2>/dev/null; then
            info "Existing daemon service uninstalled"
        else
            warn "Failed to uninstall existing daemon service (may not exist)"
        fi
    fi

    info "Installing daemon service..."
    if "$cops_bin" install; then
        info "Daemon service installed successfully"
    else
        error "Failed to install daemon service"
    fi
}

# Main installation flow
main() {
    info "C-Ops Installer"
    echo ""

    # Detect platform
    local os=$(detect_os)
    local arch=$(detect_arch)
    info "Detected platform: $os/$arch"

    # Detect SHA256 command for checksum verification
    detect_sha256_cmd

    # Check if upgrade
    local is_upgrade_install=false
    if is_upgrade; then
        is_upgrade_install=true
        info "Existing installation detected. Performing upgrade..."
    else
        info "Performing fresh installation..."
    fi

    # Get version
    local version=$(get_latest_version)
    info "Installing version: $version"

    # Download, verify, and install. Sets INSTALLED_VERSION global
    # (may differ from $version if fallback occurred).
    download_and_extract "$version" "$os" "$arch"
    version="$INSTALLED_VERSION"

    # Update PATH (only for fresh install)
    if [ "$is_upgrade_install" = false ]; then
        update_path
    fi

    # Register daemon service
    register_daemon "$is_upgrade_install"

    # Success message
    echo ""
    info "C-Ops $version installed successfully!"
    info "Installation directory: $INSTALL_DIR"
    info "Binaries: cops, cops-daemon"
    echo ""

    if [ "$is_upgrade_install" = false ]; then
        local config_file=$(get_shell_config)
        info "To use cops immediately, run:"
        echo "  source $config_file"
        echo ""
        info "Or restart your terminal."
    else
        info "Upgrade complete. You can start using the new version immediately."
    fi
}

main
