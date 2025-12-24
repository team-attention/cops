---
trigger: glob
globs: **/internal/platform/setup/**/*.go
paths: **/internal/platform/setup/**/*.go
---

# Platform Setup Guidelines

The setup package contains initialization functions for platform-level dependencies (databases, external clients, logging, etc.).

## Function Pattern

Setup functions follow these patterns:

### Root Setup (no dependencies)

```go
func Init{Service}(cfg *config.Config) *{ReturnType} {
    // Extract config, initialize, return
}
```

### Dependent Setup (requires other services)

```go
func Init{Service}(cfg *config.Config, logger *slog.Logger, ...) (*{ReturnType}, error) {
    // Use injected dependencies
    // Extract config, initialize, return
}
```

### Rules

1. **Function naming**: Always use `Init{Service}` pattern
2. **First parameter**: Always `cfg *config.Config` (full Config object)
3. **Additional parameters**: Accept required dependencies (logger, other services)
4. **Return type**: Return the initialized client/service instance
5. **Error handling**: Return error if initialization can fail

## Examples

### Logger Initialization (root, no dependencies)

```go
// internal/platform/setup/logger.go
func InitLogger(cfg *config.Config) *slog.Logger {
    var handler slog.Handler

    if cfg.Logging.DevMode {
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: parseLevel(cfg.Logging.Level),
        })
    } else {
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: parseLevel(cfg.Logging.Level),
        })
    }

    return slog.New(handler)
}
```

### Database Initialization (depends on logger)

```go
// internal/platform/setup/mongodb.go
func InitMongoDB(cfg *config.Config, logger *slog.Logger) (*mongo.Database, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    // Ping to verify connection
    if err := client.Ping(ctx, nil); err != nil {
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

    logger.Info("MongoDB initialized", slog.String("database", cfg.MongoDB.Database))
    return client.Database(cfg.MongoDB.Database), nil
}
```

## Configuration Structure

Configuration uses struct-based approach with environment variable binding:

```go
// internal/platform/setup/config/config.go
type Config struct {
    HTTP     HTTPConfig
    MongoDB  MongoDBConfig
    Logging  LoggingConfig
}

type HTTPConfig struct {
    Port int `env:"HTTP_PORT" envDefault:"8080"`
}

type MongoDBConfig struct {
    URI      string `env:"MONGODB_URI" envDefault:"mongodb://localhost:27017"`
    Database string `env:"MONGODB_DATABASE" envDefault:"alpha"`
}

type LoggingConfig struct {
    Level   string `env:"LOGGING_LEVEL" envDefault:"info"`
    DevMode bool   `env:"LOGGING_DEV_MODE" envDefault:"false"`
}
```

### Environment Variable Naming

- Use uppercase with underscores (e.g., `MONGODB_URI`, `LOGGING_LEVEL`)
- Use prefixes to group related settings (e.g., `MONGODB_`, `LOGGING_`)

### Loading Configuration

```go
func LoadConfig() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    return cfg, nil
}
```

## Container Integration

```go
// cmd/internal/container/module_platform.go
func newPlatformModule() fx.Option {
    return fx.Module("platform",
        // Config
        fx.Provide(config.LoadConfig),

        // Root setup - only takes config
        fx.Provide(setup.InitLogger),

        // Dependent setup - takes config + logger
        fx.Provide(setup.InitMongoDB),

        // More setups...
        fx.Provide(setup.InitValidator),
    )
}
```

## Best Practices

1. **Always pass full Config**: Functions receive the complete `*config.Config` object
2. **Extract what you need**: Inside the function, extract only the relevant configuration
3. **Inject dependencies**: Accept dependencies (logger, other services) as function parameters
4. **Log initialization**: Always log when a service is initialized
5. **Handle errors**: Return errors if initialization can fail
6. **Test connections**: For external services, verify the connection before returning
7. **Context with timeout**: Use context with timeout for network operations
