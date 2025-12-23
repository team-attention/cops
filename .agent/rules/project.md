# C-Ops Project Structure

This project uses a Go workspace with multiple modules.

## Directory Structure

```
cops/
├── go.work                    # Go workspace configuration
├── .claude/                   # Claude Code rules and settings
├── doc/                       # Documentation
├── idl/                       # Interface Definition Language (protobuf)
│   └── protobuf/
│       ├── buf.yaml
│       ├── buf.gen.yaml
│       └── {service}/v1/      # Service-specific protos
│
├── shared/                    # Shared module (generated code, common types)
│   ├── go.mod
│   └── gen/
│       └── grpcstub/          # Generated protobuf/ConnectRPC code
│
├── cli/                       # CLI application module
│   ├── go.mod
│   ├── cmd/                   # Entry points
│   └── internal/              # Private application code
│
├── collector/                 # (future) Collector service module
│   ├── go.mod
│   └── ...
│
└── api/                       # (future) API server module
    ├── go.mod
    └── ...
```

## Module Dependencies

```
cli        → shared (generated types)
collector  → shared (generated types)
api        → shared (generated types)
```

## Code Generation

Protobuf code is generated to `shared/gen/grpcstub/`:

```bash
cd idl/protobuf && buf generate
```

## Building

Build all modules from root:

```bash
go build ./cli/... ./shared/...
```

Or build specific module:

```bash
cd cli && go build ./...
```

## Adding New Services

1. Create proto file in `idl/protobuf/{service}/v1/{service}.proto`
2. Run `cd idl/protobuf && buf generate`
3. Import from `github.com/team-attention/cops/shared/gen/grpcstub/{service}/v1`
