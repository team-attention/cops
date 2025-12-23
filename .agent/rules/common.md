---
trigger: always_on
---

The following rules apply regardless of language or file location.

## MCP Rule
- When modifying code, do not use Serena's features; use Claude Code's tools instead.
- When using external packages, make full use of Context7 search. Before creating special functions, check how to use the package.
- **IMPORTANT**: When you need to investigate the latest usage of a package or library, you MUST use Context7 MCP to get up-to-date documentation and examples.

## Code rule
- Don't make more than what is requested. Perform only the tasks necessary to fulfill the user's request.
- All comments must be written in English.

## Dependency rule
- When adding dependencies, do not directly modify `package.json`, `pyproject.toml`, `go.mod` files, etc. Be sure to install dependencies using dependency management tool commands(ex. `npm`, `go get`, `uv`).
- Always install the latest version of the package. Use `Context7` MCP to check the latest version and how to use it before starting work.
- If there are packages available, use them as much as possible. However, those packages must be well-tested. For example, instead of writing Drag & Drop action code yourself, use a well-known Drag & Drop package that can perform the intended action.

## Architecture rule
- Choose an architecture that follows the best practices most commonly used at the framework or language level. For example, Go projects should follow https://github.com/golang-standards/project-layout to configure the top-level directory architecture, and specifically, the internal directories and architecture should follow the Hexagonal Architecture.