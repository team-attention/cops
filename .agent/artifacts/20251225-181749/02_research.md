# Research Report

## Mode
General Research

## Request Summary
Create a Makefile in the `cli/` directory with a single `make dev-build` command that compiles the CLI tool with debug symbols (`-gcflags "all=-N -l"`) and race condition detection (`-race`) enabled. The binary should be output to `cli/cops` for local development use.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
| --- | --- |
| `/Users/jayce/team-attention/cops/api/Makefile` | Reference for Makefile style and conventions in this project |
| `/Users/jayce/team-attention/cops/daemon/Makefile` | Reference for Makefile style and conventions in this project |
| `/Users/jayce/team-attention/cops/cli/cmd/cops/main.go` | Entry point file to confirm build target path |
| `/Users/jayce/team-attention/cops/cli/go.mod` | Confirms module name and Go version for build |

## Package Candidates

No external packages are required for this task. This is a Makefile creation task that uses only Go's built-in toolchain (`go build`).

## Technical Constraints

1. **Go Workspace Context**: The project uses `go.work` at the root, so builds must work within the workspace context
2. **Module Replacement**: The `cli/go.mod` has `replace github.com/team-attention/cops/shared => ../shared` which must be respected
3. **Go Version**: Project uses Go 1.25.5 (as specified in `go.work` and `cli/go.mod`)
4. **Race Detector Limitations**: The `-race` flag is only supported on certain platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64) - this is acceptable as the requirement specifies current platform only
5. **Debug Symbols**: The `-gcflags "all=-N -l"` flag disables optimizations and inlining for better debugging experience but results in slower binaries
6. **Output Location**: Binary must be output to `cli/cops` (not `cli/bin/cops` or other locations)

## Similar Implementations Found

### Example 1: api/Makefile
- **File**: `/Users/jayce/team-attention/cops/api/Makefile:1-25`
- **Relevance**: Shows Makefile conventions used in this project:
  - Uses Docker Compose for development
  - Has environment file configuration pattern
  - Uses `.PHONY` declarations
  - Uses comment-based target documentation (`## target: description`)
  - Variables are defined with `?=` for defaults

### Example 2: daemon/Makefile
- **File**: `/Users/jayce/team-attention/cops/daemon/Makefile:1-12`
- **Relevance**: Simpler Makefile example:
  - Uses Air for hot reload (different from CLI needs)
  - Shows minimal Makefile structure
  - Uses same `.PHONY` pattern
  - Uses same comment-based documentation pattern

### Key Observations from Existing Makefiles:

1. **Style Pattern**: Both use `## target: description` format for documenting targets
2. **PHONY Declaration**: Both declare `.PHONY` before each target
3. **Simplicity**: daemon/Makefile is very simple (only 12 lines), which aligns with the requirement for a simple, focused Makefile
4. **No Go Build Examples**: Neither existing Makefile shows direct `go build` commands (api uses Docker, daemon uses Air), so the CLI Makefile will be the first direct Go build example

## Go Build Flags Analysis

### Debug Symbols (`-gcflags "all=-N -l"`)
- `-N`: Disables optimizations
- `-l`: Disables inlining
- `all=`: Applies to all packages, not just the main package
- These flags make debugging with tools like `dlv` (Delve) much more effective

### Race Detector (`-race`)
- Enables the Go race detector
- Instruments the code to detect data races at runtime
- Increases binary size (~2x) and reduces performance (~2-10x slower)
- Only available on certain platforms (darwin, linux, windows on amd64/arm64)
- Appropriate for development builds only

### Build Command Structure
```bash
go build -race -gcflags "all=-N -l" -o <output> <package>
```

## CLI Directory Structure

```
cli/
├── cmd/
│   ├── cops/
│   │   └── main.go          # Entry point (build target: ./cmd/cops)
│   └── internal/
│       └── container/        # DI container setup
├── internal/
│   ├── platform/             # Platform utilities
│   └── service/              # Business logic
├── go.mod
├── go.sum
└── README.md
```

## Additional Information for Planning

1. **Entry Point**: The build command should target `./cmd/cops` (or `./cmd/cops/main.go`)
2. **Output Binary**: Should be named `cops` and placed at `cli/cops`
3. **Makefile Location**: Should be created at `/Users/jayce/team-attention/cops/cli/Makefile`
4. **No Environment Files Needed**: Unlike api and daemon, the CLI dev-build doesn't need environment file configuration
5. **Simple Structure**: The Makefile should be minimal, with only one target (`dev-build`)
6. **Consistent Documentation**: Use `## dev-build: description` format to match other Makefiles
