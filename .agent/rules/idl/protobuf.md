---
trigger: glob
globs: idl/protobuf/**/*.proto
paths: idl/protobuf/**/*.proto
---

# Protobuf Conventions

This document defines the coding standards for Protocol Buffer files in this project.

## Directory Structure

```
idl/protobuf/
├── buf.yaml           # Buf module configuration
├── buf.gen.yaml       # Code generation configuration
└── {service}/v1/      # Service-specific protos
    └── {service}.proto
```

## Package Naming

- **Package name**: `{service}.v1`
- **Go package**: `github.com/team-attention/cops/shared/gen/grpcstub/{service}/v1;{service}v1`

```protobuf
syntax = "proto3";

package collector.v1;

option go_package = "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1;collectorv1";
```

## Naming Conventions

### Messages
- Use PascalCase for message names
- Use descriptive, noun-based names

```protobuf
message SessionRecord { ... }
message SendRecordsReq { ... }
message SendRecordsRes { ... }
```

### Request/Response Suffix Convention
- Request messages: `{RPC명}Req` (NOT `{RPC명}Request`)
- Response messages: `{RPC명}Res` (NOT `{RPC명}Response`)

```protobuf
// WRONG
message SendRecordsRequest { ... }
message SendRecordsResponse { ... }

// CORRECT
message SendRecordsReq { ... }
message SendRecordsRes { ... }
```

### Fields
- Use snake_case for field names
- Protobuf compiler converts to camelCase in Go

```protobuf
message SessionRecord {
  string uuid = 1;
  string parent_uuid = 2;
  string session_id = 3;
}
```

### Services and RPCs
- Service names: `{Domain}Service`
- RPC names: PascalCase verbs

```protobuf
service CollectorService {
  rpc SendRecords(SendRecordsReq) returns (SendRecordsRes);
}
```

### Enums
- Enum type: PascalCase
- Enum values: UPPER_SNAKE_CASE with type prefix

```protobuf
enum SessionType {
  SESSION_TYPE_UNSPECIFIED = 0;
  SESSION_TYPE_USER = 1;
  SESSION_TYPE_ASSISTANT = 2;
}
```

## Field Types

### Common Types
- Use `string` for IDs
- Use `google.protobuf.Timestamp` for timestamps
- Use `repeated` for arrays
- Use `int32`/`int64` for integers
- Use `double` for floating point

```protobuf
import "google/protobuf/timestamp.proto";

message SessionRecord {
  string uuid = 1;
  google.protobuf.Timestamp timestamp = 2;
  int32 input_tokens = 3;
  double cost_usd = 4;
  repeated string tags = 5;
}
```

### Optional Fields
- In proto3, all fields are optional by default
- Use wrapper types for nullable semantics if needed

## Code Generation

### Running buf generate
```bash
cd idl/protobuf && buf generate
```

### Output Location
Generated code goes to:
- `shared/gen/grpcstub/{service}/v1/{service}.pb.go` - Protobuf types
- `shared/gen/grpcstub/{service}/v1/{service}v1connect/{service}.connect.go` - ConnectRPC handlers

### Importing Generated Code
```go
import (
    collectorv1 "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1"
    "github.com/team-attention/cops/shared/gen/grpcstub/collector/v1/collectorv1connect"
)
```

## Best Practices

1. **Reserve field numbers**: Don't reuse deleted field numbers
2. **Add comments**: Document message and field purposes
3. **Version services**: Use `/v1/`, `/v2/` for breaking changes
4. **Keep messages focused**: One purpose per message
5. **Use well-known types**: Leverage `google.protobuf.*` types

## Anti-Patterns to Avoid

### Do not use camelCase for field names
```protobuf
// WRONG
message Bad {
  string sessionId = 1;
}

// CORRECT
message Good {
  string session_id = 1;
}
```

### Do not use generic message names
```protobuf
// WRONG
message Request { ... }
message Response { ... }

// CORRECT
message SendRecordsReq { ... }
message SendRecordsRes { ... }
```

### Do not use full Request/Response suffix
```protobuf
// WRONG
message SendRecordsRequest { ... }
message SendRecordsResponse { ... }

// CORRECT
message SendRecordsReq { ... }
message SendRecordsRes { ... }
```
