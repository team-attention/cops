# Implementation Plan: Add Makefile to Shared Directory

## Overview

This plan addresses user feedback requesting a Makefile in the shared directory to provide a convenient `make test` command for running domain tests. The shared module currently has no Makefile, unlike other modules (cli, api, daemon) which all have Makefiles for common development tasks.

The task is straightforward: create a new Makefile with a `test` target that executes `go test -v ./domain/...`.

## Package Changes

None required. This is a build/development tooling addition only.

## Implementation Steps

### Step 1: Create Makefile in shared directory

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/Makefile`: Reference for comment style and .PHONY declaration pattern
- `/Users/jayce/team-attention/cops/api/Makefile`: Reference for comment style and .PHONY declaration pattern

#### `/Users/jayce/team-attention/cops/shared/Makefile`

**Description**:
Create a new Makefile with a single `test` target that runs Go tests with verbose output on the domain package. Follow the project convention of using `## target: Description` comments for self-documentation.

```makefile
## test: Run all tests in the domain package
.PHONY: test
test:
	go test -v ./domain/...
```

**Specification Details**:

| Element | Value | Rationale |
| :------ | :---- | :-------- |
| Comment format | `## test: Run all tests in the domain package` | Follows cli/Makefile pattern with `##` prefix for self-documenting targets |
| .PHONY declaration | `.PHONY: test` | Declares `test` as a phony target since it doesn't produce a file named "test" |
| Command | `go test -v ./domain/...` | `-v` for verbose output showing each test, `./domain/...` to run all tests in domain package and subpackages |
| Indentation | Tab character | Required by Makefile syntax for recipe commands |

**Test Scenarios**:

| Scenario | Command | Expected Output | Verification |
| :------- | :------ | :-------------- | :----------- |
| Run tests from shared directory | `cd /Users/jayce/team-attention/cops/shared && make test` | All domain tests run with verbose output, 32 specs pass | Exit code 0, all tests PASS |
| Run with explicit target | `make -C /Users/jayce/team-attention/cops/shared test` | Same as above | Exit code 0, all tests PASS |
| Tab shows help | `make -C /Users/jayce/team-attention/cops/shared` | Shows test target (default behavior) | Makefile is valid |

## Verification Steps

After implementation, verify the Makefile works correctly:

1. Navigate to shared directory: `cd /Users/jayce/team-attention/cops/shared`
2. Run `make test`
3. Confirm all 32 specs pass with verbose output
4. Confirm exit code is 0

## Quality Checklist

- [x] Single file to create: `/Users/jayce/team-attention/cops/shared/Makefile`
- [x] Follows project Makefile conventions (comment style, .PHONY declaration)
- [x] Uses tab indentation for recipe (Makefile requirement)
- [x] Test command matches what user was already running manually
- [x] No unnecessary additional targets beyond user request
