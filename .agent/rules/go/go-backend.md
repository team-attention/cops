---
trigger: always_on
globs: **/*.go
paths: **/*.go
---

# General Rules
- By default, new code should exhibit similarity to other pre-existing code. The response structure, parameters, etc., should be structured similarly by referencing other code.
- Backward compatibility is not a concern. This project is not a live service. When making changes, remove everything unnecessary. 

# Function Rules

## Parameters
A function accepts a maximum of three parameters, unless they are of type `context.Context`, in which case they must be preceded by a `context.Context` type. If you need more parameters, you must create a structure for the function's parameters. The structure should be named `functionNameParams` or `FunctionNameParams`. If you can write the same parameter to two functions that are part of a similar family of behaviors, you should name the parameters with a name that encompasses both functions. It should still be suffixed with `Params`.

### EXCEPTION
However, Constructor functions that should be used for dependency injection by `go.uber.org/fx` should take multiple inputs without defining a Parameter structure.

# Combined Go Style Guide
If you don't violate any of the above rules, follow the rules in the link below. If there is a conflict, follow the order of precedence from the top first

- https://google.github.io/styleguide/go/guide
- https://google.github.io/styleguide/go/decisions
- https://google.github.io/styleguide/go/best-practices
- https://go.dev/wiki/CodeReviewComments
- https://go.dev/doc/effective_go

# Project Layout
- `cmd/`: This is where your main application entry points (main functions) go.
    - Create a subdirectory for each application or service (e.g., `cmd/app/`).
    - This directory contains the code that injects dependencies and runs your app using `go.uber.org/fx`.
- `internal/`: This directory is for private application and library code.
- `pkg/`: This directory is for library code that is safe to be used by external applications.
    - Place code here that can be imported and used by other projects.
    - This code should be carefully designed and documented.
- `scripts/`: This directory is for scripts used to manage, build, install, or analyze the project.