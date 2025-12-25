# Development Walkthrough

## Summary
Created a minimal development build Makefile for the CLI module with debug symbols and race detection enabled, allowing developers to build the `cops` binary locally for debugging purposes.

## Code Overview

### New Components

#### `cli/Makefile`
- **Location**: `/Users/jayce/team-attention/cops/cli/Makefile`
- **Purpose**: Provides a single `dev-build` target for local CLI development builds
- **Key Features**:
  - **Debug Symbols**: Uses `-gcflags "all=-N -l"` to disable optimizations and inlining, making the binary debugger-friendly
  - **Race Detection**: Includes `-race` flag to enable Go's race detector for finding concurrency bugs
  - **Output Location**: Builds binary to `cli/cops` (directly in the cli directory)
  - **Build Target**: Compiles from `./cmd/cops` entry point

**Makefile Content**:
```makefile
## dev-build: Build CLI with debug symbols and race detection
.PHONY: dev-build
dev-build:
	go build -race -gcflags "all=-N -l" -o cops ./cmd/cops
```

**Design Rationale**:
- Follows the minimal style of `daemon/Makefile` (12 lines) rather than the more complex `api/Makefile` (25 lines with Docker)
- Uses consistent documentation format (`## target: description`) matching other project Makefiles
- No environment file configuration needed (unlike api/daemon) since this is a simple build command
- Single target keeps the Makefile focused and maintainable

### Modified Components

#### `.gitignore`
- **Location**: `/Users/jayce/team-attention/cops/.gitignore`
- **Changes**: Added `cli/cops` to prevent tracking the development binary
- **Rationale**: Development builds should not be committed to version control; only the source code and Makefile are tracked

```diff
+cli/cops
```

## Testing

### Build Verification
All verification commands were executed successfully:

```bash
# Build all modules
go build ./cli/... ./daemon/...     # Result: PASS

# Build entire workspace
go build ./...                       # Result: PASS

# Run dev-build target
cd cli && make dev-build            # Result: PASS
# Output: Binary created at cli/cops (36MB with race detector)

# Verify binary executes
./cli/cops --help                   # Result: PASS
# Output: CLI help text displayed correctly
```

### Binary Characteristics
- **Size**: 36MB (significantly larger than optimized builds due to race detector instrumentation)
- **Race Detection**: Enabled (increases binary size ~2x and reduces performance ~2-10x)
- **Debug Symbols**: Present (disables optimizations for better debugging with `dlv`)
- **Platform**: Built for current development platform only (darwin/arm64)

### Test Coverage
- No unit tests required (Makefile-only change)
- Manual testing verified:
  - Makefile syntax is correct
  - Build command executes without errors
  - Binary runs and displays help output
  - Binary is excluded from git tracking

## Issues & Resolutions

| Issue | Resolution |
|-------|------------|
| Binary size concerns (36MB) | Expected behavior - race detector adds significant instrumentation. This is acceptable for development builds as the tradeoff provides better debugging capabilities. |
| Consistency with existing Makefiles | Followed daemon/Makefile minimal style rather than api/Makefile Docker style, as CLI dev-build is a simple compilation task without runtime dependencies. |

## Key Decisions

### 1. Build Flags Selection
**Decision**: Use `-race -gcflags "all=-N -l"` combination
**Rationale**:
- `-race`: Catches data races at runtime during development testing
- `-gcflags "all=-N -l"`: Makes debugging with Delve more effective by preserving variable names and preventing inlining
- `all=`: Applies flags to all packages, not just main package

### 2. Output Location
**Decision**: Output to `cli/cops` instead of `cli/bin/cops` or other nested location
**Rationale**:
- Keeps development workflow simple (`./cops --help` instead of `./bin/cops --help`)
- Matches user requirements exactly
- Easier to add to `.gitignore` with specific path

### 3. Single Target Only
**Decision**: Only implement `dev-build` target, no test/lint/clean targets
**Rationale**:
- Requirements specified only dev-build was needed
- Keeps Makefile minimal and focused
- Additional targets can be added later if needed

## Build Command Breakdown

```bash
go build -race -gcflags "all=-N -l" -o cops ./cmd/cops
```

| Flag | Purpose | Impact |
|------|---------|--------|
| `-race` | Enable race detector | +~100% binary size, detects concurrency bugs at runtime |
| `-gcflags "all=-N -l"` | Disable optimizations (`-N`) and inlining (`-l`) for all packages | Better debugging experience, slower execution |
| `-o cops` | Output binary to `cli/cops` | Custom output location |
| `./cmd/cops` | Build target | Entry point at `cmd/cops/main.go` |

## Usage

### Building the CLI
```bash
cd cli/
make dev-build
```

### Running the CLI
```bash
./cops --help           # Show help
./cops add [directory]  # Register a project
./cops list            # List registered projects
```

### Debugging with Delve
```bash
# The debug symbols make this effective:
dlv exec ./cops -- list
```

## Related Tickets
- Task: Create CLI Makefile with dev-build target
- Acceptance Criteria: All 7 criteria met (Makefile location, debug flags, race detection, output location, entry point, platform targeting, consistency)

## Notes for Future Development

1. **Production Builds**: When production builds are needed, create a separate `build` target without `-race` and with optimizations enabled (`-ldflags "-s -w"` for smaller binaries)

2. **Cross-Platform Builds**: The current dev-build targets the current platform only. For multi-platform releases, consider adding targets like:
   ```makefile
   build-linux:
       GOOS=linux GOARCH=amd64 go build -o cops-linux-amd64 ./cmd/cops
   ```

3. **Version Injection**: Production builds should inject version info using `-ldflags`:
   ```makefile
   -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"
   ```

4. **Clean Target**: Consider adding if binary cleanup is needed:
   ```makefile
   clean:
       rm -f cops
   ```
