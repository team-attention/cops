# add-project

## Intent

Provide a guided way for Claude Code users to register their project for C-Ops session tracking.

## Motivation

The `cops add` command uses an interactive TUI (bubbletea) for project configuration, including
git detection, naming, organization selection, and sync preferences. Since this requires terminal
interaction, the skill guides the user through the process rather than executing it directly.

## Design Decisions

- **User-executed command**: The TUI requires interactive terminal input, so Claude constructs the command and the user runs it
- **Pre/post verification**: Checks cops installation before and project registration after
- **Minimal parameters**: Only `DIR` and `NO_GIT` since other options are configured interactively in the TUI

## Constraints

- Requires cops to be installed (`/install-cops` first)
- Requires interactive terminal (TUI cannot run in piped/non-TTY environment)
- Requires authentication (organization selection needs valid auth)
