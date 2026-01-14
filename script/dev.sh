#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="$HOME/.cops/bin"
REPO_OWNER="team-attention"
REPO_NAME="cops"

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

# Check requirements
check_requirements() {
    if ! command -v gh &> /dev/null; then
        error "gh CLI is required. Install: https://cli.github.com/"
    fi
    if ! gh auth status &> /dev/null; then
        error "GitHub authentication required. Run: gh auth login"
    fi

    # Verify access to the repository
    info "Verifying repository access..."
    if ! gh repo view "$REPO_OWNER/$REPO_NAME" &> /dev/null; then
        warn "Cannot access $REPO_OWNER/$REPO_NAME with current account."
        info "Please select an account with access to team-attention org:"
        echo "" >&2

        # Run interactive account switch
        if ! gh auth switch; then
            error "Failed to switch account. Run 'gh auth login' with an account that has access."
        fi

        # Retry access check after switch
        info "Retrying repository access..."
        if ! gh repo view "$REPO_OWNER/$REPO_NAME" &> /dev/null; then
            error "Still cannot access $REPO_OWNER/$REPO_NAME. Please login with an authorized account: gh auth login"
        fi
    fi
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

# Get latest version using gh CLI
get_latest_version() {
    local version="${COPS_VERSION:-}"

    if [ -z "$version" ]; then
        info "Fetching latest version from GitHub..."
        version=$(gh release view --repo "$REPO_OWNER/$REPO_NAME" --json tagName -q '.tagName' 2>/dev/null)

        if [ -z "$version" ]; then
            error "Failed to fetch latest version. Make sure you have access to the repository."
        fi
    fi

    echo "$version"
}

# Check if this is an upgrade
is_upgrade() {
    [ -f "$INSTALL_DIR/cops" ] || [ -f "$INSTALL_DIR/cops-daemon" ]
}

# Download and extract archive using gh CLI
download_and_extract() {
    local version=$1
    local os=$2
    local arch=$3

    local archive_name="${REPO_NAME}_${version#v}_${os}_${arch}.tar.gz"
    local tmp_dir=$(mktemp -d)

    info "Downloading $archive_name..."
    if ! gh release download "$version" \
        --repo "$REPO_OWNER/$REPO_NAME" \
        --pattern "$archive_name" \
        --dir "$tmp_dir"; then
        rm -rf "$tmp_dir"
        error "Failed to download $archive_name"
    fi

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

    # Cleanup
    rm -rf "$tmp_dir"
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
    info "C-Ops Installer (Dev)"
    echo ""

    # Check requirements first
    check_requirements

    # Detect platform
    local os=$(detect_os)
    local arch=$(detect_arch)
    info "Detected platform: $os/$arch"

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

    # Download and install
    download_and_extract "$version" "$os" "$arch"

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
