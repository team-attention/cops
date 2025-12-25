# Implementation Plan

## Overview
Create a Makefile in the `cli/` directory with a single `dev-build` target that compiles the CLI tool with debug symbols and race detection for local development.

## Selected Packages

| Problem | Package | Context7 ID | Reason for Selection |
| ------- | ------- | ----------- | -------------------- |
| N/A | N/A | N/A | No external packages required - uses Go's built-in toolchain only |

## Architecture Decisions

### Decision 1: Makefile Structure
**Choice**: Create a minimal Makefile following the daemon/Makefile style (simple, focused) rather than the api/Makefile style (Docker-based with environment variables).
**Rationale**: The CLI dev-build is a direct Go compilation task that doesn't require Docker, environment files, or complex configuration. The daemon/Makefile structure (11 lines) is more appropriate than the api/Makefile structure (25 lines).

### Decision 2: Build Command Structure
**Choice**: Use `go build -race -gcflags "all=-N -l" -o cops ./cmd/cops`
**Rationale**:
- `-race`: Enables race detection (required by request)
- `-gcflags "all=-N -l"`: Disables optimizations and inlining for debugging (required by request)
- `-o cops`: Outputs to `cli/cops` (required by request)
- `./cmd/cops`: Targets the entry point at `cmd/cops/main.go`

### Decision 3: No Environment Files
**Choice**: Do not include environment file configuration (unlike api/ and daemon/ Makefiles).
**Rationale**: The dev-build target is a simple compilation command that doesn't require runtime configuration. Adding environment files would add unnecessary complexity.

### Decision 4: Documentation Format
**Choice**: Use `## target: description` comment format before `.PHONY` declaration.
**Rationale**: Matches the existing convention in both api/Makefile and daemon/Makefile for consistency.

## Implementation Steps

### Step 1: Create cli/Makefile

**Files to Create/Modify**:
- `/Users/jayce/team-attention/cops/cli/Makefile` (create)

**Content**:

```makefile
## dev-build: Build CLI with debug symbols and race detection
.PHONY: dev-build
dev-build:
	go build -race -gcflags "all=-N -l" -o cops ./cmd/cops
```

**Test Scenarios**:
| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Successful build | `make dev-build` in cli/ | Binary created at `cli/cops` | Happy path |
| Run built binary | `./cops --help` | CLI help output displayed | Verify binary is functional |
| Race detector present | `file cops` or run with race-enabled code | Binary includes race instrumentation | Race flag verification |

## Execution Order

1. Step 1 (no dependencies) - Create the Makefile

## Notes for Execute Agent
- The Makefile must be created at `/Users/jayce/team-attention/cops/cli/Makefile`
- Ensure there is a newline at the end of the file
- The command uses a tab character (not spaces) for indentation in the recipe
- After creation, verify by running `make dev-build` from the `cli/` directory
- The output binary `cli/cops` should be added to `.gitignore` if not already present (optional - check existing .gitignore patterns)
