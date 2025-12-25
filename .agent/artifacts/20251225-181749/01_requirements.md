# Requirements

## Request Summary
Create a Makefile in the `cli/` directory with a single development build command (`make dev-build`) that compiles the CLI tool with debug symbols and race condition detection enabled. The binary should be output to `cli/cops` for local development use.

## Acceptance Criteria

- [ ] Criterion 1: Makefile is created at `/Users/jayce/team-attention/cops/cli/Makefile`
- [ ] Criterion 2: `make dev-build` command builds the binary with debug symbols enabled (`-gcflags "all=-N -l"`)
- [ ] Criterion 3: `make dev-build` command includes race detector flag (`-race`)
- [ ] Criterion 4: Built binary is output to `cli/cops` (not `cli/bin/cops` or other location)
- [ ] Criterion 5: Build command compiles from `cmd/cops/main.go` entry point
- [ ] Criterion 6: Build targets only the current development platform (no cross-compilation)
- [ ] Criterion 7: Makefile follows the pattern used in `api/Makefile` and `daemon/Makefile` (consistent style)

## Scope

### In Scope
- Single build target: `make dev-build`
- Debug symbols inclusion via `-gcflags "all=-N -l"`
- Race condition detection via `-race` flag
- Output binary to `cli/cops` location
- Build from `cmd/cops/main.go` entry point
- Support for current platform only (GOOS/GOARCH not specified)

### Out of Scope
- Hot reload functionality (not needed, unlike daemon)
- Test commands (`make test`, `make coverage`, etc.)
- Cross-platform builds (macOS, Linux, Windows)
- Multiple architecture support (ARM64, AMD64)
- Production/release builds
- Installation commands (`make install`)
- Clean commands (`make clean`)
- Dependency management commands (`make deps`, `make tidy`)
- Code quality tools (`make lint`, `make fmt`)
- Run commands (`make run`)
- Help command (`make help`)
- Version injection via ldflags
- Environment file configuration
- Docker-based builds

## Constraints
- Build must work in Go workspace context (uses `go.work`)
- Build must respect local module replacement (`replace github.com/team-attention/cops/shared => ../shared`)
- Makefile should be simple and focused (only one target needed)
- Debug build should be slower but provide better debugging experience
- Race detector will increase binary size and reduce performance (acceptable for dev builds)

## Additional Context
- CLI module location: `/Users/jayce/team-attention/cops/cli/`
- Entry point: `cmd/cops/main.go`
- Go version: 1.25.5
- Project uses Go workspace (`go.work` at root)
- Existing Makefiles for reference:
  - `api/Makefile`: Uses Docker Compose, environment files
  - `daemon/Makefile`: Uses Air for hot reload
- CLI uses `uber.go/dig` for dependency injection
- CLI commands: `cops add`, `cops list` (from README.md)

## Questions Resolved

| Question | Answer |
| --- | --- |
| Which build commands are needed? | Only `make dev-build` |
| Should debug symbols be included? | Yes, use `-gcflags "all=-N -l"` |
| Should race detector be enabled? | Yes, use `-race` flag |
| Should version info be injected? | No, not needed for dev builds |
| Where should the binary be output? | `cli/cops` (directly in cli directory) |
| Is hot reload needed? | No, manual build only |
| Are test commands needed? | No |
| Is cross-platform build needed? | No, current platform only |
| Are dependency management commands needed? | No |
| Are other dev tools needed (lint, fmt)? | No |
