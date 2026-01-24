# install-cops

## Intent

Provide a one-command way for Claude Code users to install or upgrade the C-Ops CLI tool.

## Motivation

C-Ops requires downloading platform-specific binaries and configuring PATH and daemon services.
This skill encapsulates the entire installation process so users can simply invoke `/install-cops`
without needing to know the install script URL or manual setup steps.

## Design Decisions

- **No bundled script**: Uses the install script from the main branch via curl, ensuring users always get the latest installer logic
- **Optional version pinning**: Supports `VERSION` parameter for reproducible installs
- **No scripts/ directory**: The install script lives in the main repo (`script/install.sh`), not duplicated in the plugin

## Constraints

- Requires internet access to download from GitHub
- Only supports macOS (darwin) and Linux
- Only supports amd64 and arm64 architectures
- Requires curl to be available
