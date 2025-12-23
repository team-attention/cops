---
trigger: always_on
globs: **/internal/service/**/*.go, **/internal/service/*.go
paths: **/internal/service/**/*.go, **/internal/service/*.go
---

# Service Package Guidelines

The service package contains all business logic organized by domain following hexagonal architecture.

## Service Directory Structure

```
{domain}/
├── {domain}_service.go         # Core service implementation
├── inbound/                    # Inbound adapters - see go-inbound.md
│   └── {category}/             # Protocol category (grpc, http, worker)
│       └── {implementation}/   # Implementation (connectrpc, fiber, queue)
└── outbound/                   # Outbound adapters - see go-outbound.md
    └── {category}/             # Dependency category (repository, etc.)
        ├── {name}_port.go      # Interface (Port)
        └── {implementation}/   # Implementation (mongodb, etc.)
```

**For detailed inbound structure rules, see [go-inbound.md](./go-inbound.md)**
**For detailed outbound structure rules, see [go-outbound.md](./go-outbound.md)**

## Structure
```go
type Service struct {
    logger *slog.Logger
    // Dependencies injected via constructor
}

func NewService(l *slog.Logger, /* Dependencies */) *Service {
    return &Service{
        logger: l.With(slog.String("name", "{domain}.service")),
        /* Dependencies */
    }
}
```

### Method Pattern
1. Check RBAC permissions first
2. Validate business logic
3. Execute operation
4. Return result

```go
func (s *Service) CreateAgent(ctx context.Context, resource structure.Resource, agent domain.Agent) (*domain.Agent, error) {
    // 1. Check permissions
    if err := s.rbacSvc.CanWriteAgent(ctx, resource); err != nil {
        return nil, err
    }
    
    // 2. Business logic
    return s.repo.Create(ctx, agent)
}
```