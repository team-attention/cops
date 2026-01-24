# remove-cops

## Intent

Provide a clean way to fully uninstall C-Ops from a user's system.

## Motivation

When users want to remove C-Ops, they need to stop the daemon service, clean up shell PATH modifications, and remove the installation directory. This skill handles all cleanup steps in the correct order so nothing is left behind.

## Design Decisions

- **Graceful daemon stop**: Attempts `cops uninstall` first but continues if it fails (binary may already be deleted)
- **Shell config cleanup**: Removes both the PATH export and the installer comment, matching exactly what the installer adds
- **Complete removal**: Deletes `~/.cops/` entirely including config, auth, and socket files
- **No confirmation prompt**: The skill description clearly states it's irreversible; the user invoking it is confirmation enough

## Constraints

- Only cleans shell configs that the installer would have modified (zshrc, zprofile, bash_profile, bashrc, profile)
- Does not remove project-level `.cops/` directories (those are per-repo config)
