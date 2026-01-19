---
sidebar_position: 1
---

# Installation Guide

This guide explains how to install the C-Ops CLI and Daemon.

## Prerequisites

Before installation, ensure the following:

- **GitHub CLI Authentication**: The installation script uses GitHub API to download the latest release

## Installation via Homebrew (macOS/Linux)

The easiest way to install is using Homebrew:

```bash
# Add tap
brew tap team-attention/cops

# Install
brew install cops
```

**Benefits of Homebrew installation:**
- Automatic updates (`brew upgrade cops`)
- Easy version management
- Automatic dependency handling

## Automatic Installation Script (Recommended)

Install the CLI and Daemon service with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | bash
```

### What the Script Does

The installation script automatically handles:

1. **Platform detection**: Automatically detects your OS and architecture
2. **Binary download**: Downloads the latest version from GitHub Releases
3. **Installation**: Installs binaries to the `~/.cops/bin/` directory
4. **PATH setup**: Adds the PATH environment variable to your shell config
5. **Daemon service registration**: Automatically registers and starts the background Daemon service

## Direct Binary Download

You can also download binaries directly from the GitHub Releases page:

1. Visit the [GitHub Releases](https://github.com/team-attention/cops/releases) page
2. Download the appropriate file for your platform:
   - macOS Intel: `cops_*_darwin_amd64.tar.gz`
   - macOS Apple Silicon: `cops_*_darwin_arm64.tar.gz`
   - Linux x86_64: `cops_*_linux_amd64.tar.gz`
   - Linux ARM64: `cops_*_linux_arm64.tar.gz`
3. Extract and place the binaries in your preferred location
4. Add the binary path to your PATH

```bash
# Example: Manual installation after download
tar -xzf cops_*_darwin_arm64.tar.gz
mkdir -p ~/.cops/bin
mv cops cops-daemon ~/.cops/bin/
export PATH="$HOME/.cops/bin:$PATH"
```

## Supported Platforms

| OS | Architecture | Support |
|----|--------------|---------|
| macOS | Intel (x86_64) | Yes |
| macOS | Apple Silicon (arm64) | Yes |
| Linux | x86_64 | Yes |
| Linux | ARM64 | Yes |

## Installing a Specific Version

To install a specific version, use the `COPS_VERSION` environment variable:

```bash
COPS_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/team-attention/cops/main/script/install.sh | bash
```

## Post-Installation Verification

After installation, restart your terminal or reload your shell config:

```bash
# For zsh users
source ~/.zprofile

# For bash users
source ~/.bash_profile
```

Verify the installation:

```bash
# Check CLI version
cops --help

# Check Daemon version
cops-daemon --version
```

## Next Steps

Once installation is complete, follow the [Quick Start](./quick-start) guide to register your first project.
