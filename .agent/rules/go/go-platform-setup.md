---
trigger: glob
globs: **/internal/platform/setup/*.go
paths: **/internal/platform/setup/*.go
---

# Platform Setup Guidelines

The setup package contains initialization functions for platform-level dependencies (databases, external clients, logging, etc.).

## Package Structure

All setup files are placed directly in the `setup` package (not in subdirectories):

```
platform/setup/
├── config.go      # Configuration loading
├── logger.go      # Logger initialization
├── sqlite.go      # SQLite database initialization
├── mongodb.go     # MongoDB database initialization (example)
└── copsapi.go     # API client initialization
```

**Pattern**: `setup/{resource}.go` - All in one package (`package setup`)

## Function Pattern

Setup functions follow these patterns:

### Root Setup (no dependencies)

```go
func Init{Service}(cfg *Config) *{ReturnType} {
    // Extract config, initialize, return
}
```

### Dependent Setup (requires other services)

```go
func Init{Service}(cfg *Config, logger *slog.Logger, ...) (*{ReturnType}, error) {
    // Use injected dependencies
    // Extract config, initialize, return
}
```

### Rules

1. **Function naming**: Always use `Init{Service}` pattern
2. **First parameter**: Always `cfg *Config` (full Config object, same package)
3. **Additional parameters**: Accept required dependencies (logger, other services)
4. **Return type**: Return the initialized client/service instance
5. **Error handling**: Return error if initialization can fail

## Examples

### Logger Initialization (root, no dependencies)

```go
// internal/platform/setup/logger.go
package setup

func InitLogger(cfg *Config) *slog.Logger {
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
package setup

func InitMongoDB(cfg *Config, logger *slog.Logger) (*mongo.Database, error) {
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
// internal/platform/setup/config.go
package setup

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
import "github.com/team-attention/cops/daemon/internal/platform/setup"

func newPlatformModule() fx.Option {
    return fx.Module("platform",
        // Configuration (root - no dependencies)
        fx.Provide(setup.LoadConfig),

        // Logger (depends on config)
        fx.Provide(setup.InitLogger),

        // SQLite DB (depends on config and logger)
        fx.Provide(setup.InitSQLite),

        // API Client (depends on config)
        fx.Provide(setup.InitAPIClient),
    )
}
```

## Table Creation Pattern

Database setup functions should handle table/schema creation:

```go
func InitSQLite(cfg *Config, logger *slog.Logger) (*sql.DB, error) {
    // ... connection setup ...

    // Create tables as part of initialization
    if err := createTables(db); err != nil {
        db.Close()
        return nil, fmt.Errorf("failed to create tables: %w", err)
    }

    return db, nil
}

// Helper function for table creation
func createTables(db *sql.DB) error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS table_name (...)`,
        // Add more tables as needed
    }

    for _, query := range queries {
        if _, err := db.Exec(query); err != nil {
            return err
        }
    }
    return nil
}
```

**Why here and not in adapters?**
- Infrastructure setup is platform concern
- Shared across all adapters using the DB
- Centralized schema management

## Best Practices

1. **Single package**: All setup files in `platform/setup/` package (no subdirectories)
2. **Always pass full Config**: Functions receive the complete `*Config` object
3. **Extract what you need**: Inside the function, extract only the relevant configuration
4. **Inject dependencies**: Accept dependencies (logger, other services) as function parameters
5. **Log initialization**: Always log when a service is initialized
6. **Handle errors**: Return errors if initialization can fail
7. **Test connections**: For external services, verify the connection before returning
8. **Context with timeout**: Use context with timeout for network operations
9. **Create tables/schemas**: Database setup functions should handle table creation
