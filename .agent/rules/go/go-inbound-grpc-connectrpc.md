---
trigger: glob
globs: **/internal/service/*/inbound/grpc/connectrpc/*.go
paths: **/internal/service/*/inbound/grpc/connectrpc/*.go
---

# ConnectRPC Inbound Handler Guidelines

This document defines guidelines specific to ConnectRPC implementation of gRPC handlers.

**For common patterns, see:**
- [go-inbound.md](./go-inbound.md) - Handler structure, module registration
- [go-logging-conventions.md](./go-logging-conventions.md) - Logger binding

## ConnectHandler Interface

```go
type ConnectHandler interface {
    GetHandler(opts ...connect.HandlerOption) (string, http.Handler)
}
```

## Handler Implementation

```go
package connectrpc

import (
    "log/slog"
    "net/http"

    "connectrpc.com/connect"
    {domain}v1 "github.com/.../gen/grpcstub/{domain}/v1"
    "{domain}v1connect"
    "{domain}"
)

type {Domain}GRPCHandler struct {
    svc    *{domain}.Service
    logger *slog.Logger
}

func New{Domain}GRPCHandler(l *slog.Logger, svc *{domain}.Service) *{Domain}GRPCHandler {
    return &{Domain}GRPCHandler{
        svc:    svc,
        logger: l.With(slog.String("name", "{domain}.grpc.connectrpc")),
    }
}

// GetHandler implements ConnectHandler interface
func (h *{Domain}GRPCHandler) GetHandler(opts ...connect.HandlerOption) (string, http.Handler) {
    return {domain}v1connect.New{Domain}ServiceHandler(h, opts...)
}

// Interface verification
var _ {domain}v1connect.{Domain}ServiceHandler = (*{Domain}GRPCHandler)(nil)
```

## RPC Method Pattern

```go
func (h *{Domain}GRPCHandler) {MethodName}(
    ctx context.Context,
    req *connect.Request[{domain}v1.{MethodName}Req],
) (*connect.Response[{domain}v1.{MethodName}Res], error) {
    // 1. Parse request
    params := req.Msg

    // 2. Call service
    result, err := h.svc.{ServiceMethod}(ctx, params)
    if err != nil {
        return nil, err
    }

    // 3. Convert to protobuf response
    res := &{domain}v1.{MethodName}Res{
        // Map result to protobuf fields
    }

    return connect.NewResponse(res), nil
}
```

## Key Requirements

- **Package name**: `connectrpc`
- **Logger name**: `"{domain}.grpc.connectrpc"`
- **Interface verification**: `var _ {Service}Handler = (*Handler)(nil)`
- **Import proto**: `{domain}v1 "github.com/.../gen/grpcstub/{domain}/v1"`
