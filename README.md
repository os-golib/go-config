# Go Configuration Library

A powerful, feature-rich configuration management library for Go applications with support for multiple sources, validation, encryption, templating, and more.

## Features

- **Multi-source Support**: Load configuration from files (JSON/YAML), environment variables, memory, glob patterns, and custom sources
- **Fluent Builder API**: Clean, chainable API for building complex configurations
- **Validation**: Built-in validation using `go-playground/validator/v10` with custom rules
- **Type-safe Access**: Type-safe getters for strings, integers, booleans, slices, etc.
- **Nested Struct Binding**: Automatic binding of configuration to nested structs
- **Template Processing**: Support for Go templates in configuration values
- **Encryption**: AES-GCM encryption for sensitive values
- **Caching & Retry**: Configurable caching and retry logic for sources
- **Profiles**: Environment/profile-based configuration management
- **Lifecycle Hooks**: Extensible hook system for pre/post processing
- **Hot Reloading**: Watch for configuration changes and auto-reload
- **Middleware**: Chainable source middleware for caching, encryption, templating, etc.
- **Priority-based Merging**: Higher priority sources override lower ones
- **Type Converters**: Customizable type conversion system

## Quick Start

### Installation

```bash
go get github.com/os-golib/go-config
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/os-golib/go-config"
)

func main() {
    // Create a simple configuration
    cfg := config.NewBuilder().
        AddFile("config.yaml").
        AddEnv("APP_").
        MustBuild()

    // Access values
    port := cfg.GetInt("server.port", 8080)
    debug := cfg.GetBool("debug", false)
    hosts := cfg.GetStringSlice("hosts", []string{"localhost"})
    
    fmt.Printf("Server port: %d\n", port)
}
```

### Fluent Builder Pattern

```go
builder := config.NewBuilder().
    WithContext(ctx).
    WithDefaultPriority(10).
    WithTemplateProcessing().
    WithCaching(5*time.Minute).
    WithRetry(3, time.Second).
    AddFile("config.yaml").
    AddEnv("APP_").
    AddMemory(map[string]any{
        "defaults.env": "production",
    })
```

### Configuration Profiles

```go
cfg := config.NewBuilder().
    EnableProfiles().
    AddProfile("development", map[string]any{
        "debug": true,
        "log_level": "debug",
    }).
    AddProfile("production", map[string]any{
        "debug": false,
        "log_level": "info",
    }).
    SetActiveProfile("development").
    MustBuild()
```

### Struct Binding with Validation

```go
type AppConfig struct {
    App struct {
        Name    string `config:"name" validate:"required,min=3"`
        Version string `config:"version" validate:"required,semver"`
    } `config:"app"`
    
    Server struct {
        Host string `config:"host" validate:"required,hostname"`
        Port int    `config:"port" validate:"required,min=1,max=65535"`
    } `config:"server"`
    
    Database struct {
        URL string `config:"url" validate:"required,url"`
    } `config:"database"`
}

func main() {
    cfg := config.NewBuilder().
        AddFile("config.yaml").
        MustBuild()
    
    var appConfig AppConfig
    if err := cfg.BindAndValidate(&appConfig); err != nil {
        panic(err)
    }
    
    fmt.Printf("App: %s v%s\n", appConfig.App.Name, appConfig.App.Version)
}
```

### Pre-configured Builders

```go
// Development configuration
devCfg := config.NewDevelopmentConfig().
    AddFile("config.dev.yaml").
    MustBuild()

// Production configuration  
prodCfg := config.NewProductionConfig().
    AddFile("/etc/app/config.yaml").
    MustBuild()

// Test configuration
testCfg := config.NewTestConfig().
    MustBuild()
```

## Configuration Sources

### File Sources

```go
// JSON or YAML files
builder.AddFile("config.json")
builder.AddFile("config.yaml")

// Multiple files
builder.AddFiles("base.yaml", "overrides.yaml")

// Glob patterns
builder.AddGlob("config/*.yaml")
```

### Environment Variables

```go
// With prefix
builder.AddEnv("APP_")

// All environment variables
builder.AddEnv("")
```

### Memory Sources

```go
builder.AddMemory(map[string]any{
    "server.host": "localhost",
    "server.port": 8080,
    "features.enabled": true,
})
```

### Composite Sources

```go
builder.AddComposite("overrides", 100,
    config.Memory(map[string]any{"key": "value"}),
    config.File("overrides.yaml"),
)
```

### Conditional Sources

```go
builder.AddConditional(
    config.File("secrets.yaml"),
    func() bool { return os.Getenv("ENVIRONMENT") == "production" },
)
```

## Validation Rules

### Built-in Rules

```go
// Fluent validation API
builder.AddRules(
    config.Rules.Required("api.key"),
    config.Rules.Range("server.port", 1, 65535),
    config.Rules.Email("admin.email"),
    config.Rules.URL("api.endpoint"),
    config.Rules.Min("max_connections", 10),
    config.Rules.Max("timeout", 30),
    config.Rules.OneOf("environment", "dev", "staging", "prod"),
    config.Rules.Pattern("username", "^[a-zA-Z0-9_]+$"),
)

// Custom validator rules
builder.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
    return semver.IsValid(fl.Field().String())
})
```

## Middleware

### Caching

```go
// Cache source results for 5 minutes
builder.WithCaching(5 * time.Minute)
```

### Retry Logic

```go
// Retry failed loads with exponential backoff
builder.WithRetry(3, time.Second)
```

### Template Processing

```go
// Enable Go template processing
builder.WithTemplateProcessing()

// Add custom template functions
builder.AddTemplateFunction("getEnv", os.Getenv)
builder.AddTemplateFunction("add", func(a, b int) int { return a + b })
```

### Encryption

```go
// Enable AES-GCM encryption with prefix
builder.WithEncryption("my-secret-key")

// In config file:
// database.password: "ENC:encrypted-base64-string"
```

## Lifecycle Hooks

```go
// Logging hook
builder.AddLoggingHook(myLogger)

// Validation hook
builder.AddValidationHook(func(data map[string]any) error {
    if val, ok := data["required_key"]; !ok || val == "" {
        return fmt.Errorf("required_key is missing")
    }
    return nil
})

// Defaults hook
builder.AddDefaultsHook(map[string]any{
    "log_level": "info",
    "timeout":   30,
})
```

## Type Converters

```go
// Register custom type converters
builder.RegisterTypeConverter(reflect.TypeOf(time.Duration(0)), 
    func(dst reflect.Value, raw any) error {
        s := fmt.Sprint(raw)
        d, err := time.ParseDuration(s)
        if err != nil {
            return err
        }
        dst.Set(reflect.ValueOf(d))
        return nil
    })

// Built-in converters for:
// - time.Duration
// - url.URL
// - All primitive types
// - Slices
// - Nested structs
```

## Watching for Changes

```go
// Watch for file changes and auto-reload
cfg, err := builder.BuildAndWatch(10 * time.Second)
if err != nil {
    panic(err)
}

// Add observer for change notifications
cfg.ObserveFunc(func(changed map[string]any) {
    fmt.Printf("Configuration changed: %v\n", changed)
})

// Graceful shutdown
defer cfg.Close()
```

## Advanced Usage

### Custom Sources

```go
type CustomSource struct {
    config.BaseSource
}

func NewCustomSource(priority int) *CustomSource {
    return &CustomSource{
        BaseSource: config.NewBaseSource("custom", priority),
    }
}

func (s *CustomSource) Load() (map[string]any, error) {
    // Implement your custom loading logic
    return map[string]any{
        "custom.key": "value",
    }, nil
}

// Usage
builder.AddSource(NewCustomSource(50))
```

### Custom Middleware

```go
func WithLogging(logger Logger) config.SourceMiddleware {
    return func(src config.Source) config.Source {
        return &LoggedSource{
            BaseSource: config.NewBaseSource("logged:"+src.Name(), src.Priority()),
            source:     src,
            logger:     logger,
        }
    }
}

// Usage
builder.WithMiddleware(WithLogging(myLogger))
```

### Multi-environment Configuration

```go
func createConfig(env string) *config.Config {
    builder := config.NewBuilder().
        AddFile("config/base.yaml").
        AddFile(fmt.Sprintf("config/%s.yaml", env)).
        AddEnv("APP_")
    
    switch env {
    case "development":
        builder.WithTemplateProcessing()
    case "production":
        builder.WithCaching(5*time.Minute).
            WithRetry(3, time.Second)
    }
    
    return builder.MustBuild()
}
```

## Configuration File Examples

### YAML Configuration

```yaml
# config.yaml
app:
  name: "My Application"
  version: "1.0.0"
  env: "{{ env "APP_ENV" | default "development" }}"

server:
  host: "0.0.0.0"
  port: 8080
  timeout: "30s"

database:
  host: "{{ env "DB_HOST" }}"
  port: 5432
  name: "{{ env "DB_NAME" }}"
  username: "{{ env "DB_USER" }}"
  password: "ENC:{{ env "DB_ENCRYPTED_PASSWORD" }}"

logging:
  level: "info"
  format: "json"

features:
  cache_enabled: true
  api_enabled: true
  max_connections: 100

profiles:
  development:
    debug: true
    log_level: "debug"
  production:
    debug: false
    log_level: "warn"
```

### Environment Variables

```bash
# Shell environment
export APP_ENV=production
export DB_HOST=localhost
export DB_NAME=mydb
export DB_USER=admin
export DB_ENCRYPTED_PASSWORD=base64-encrypted-string
```

## Best Practices

1. **Use Struct Binding**: Always bind configuration to typed structs for compile-time safety
2. **Validate Early**: Validate configuration as soon as it's loaded
3. **Use Profiles**: Leverage profiles for different environments
4. **Secure Sensitive Data**: Use encryption for passwords and secrets
5. **Watch for Changes**: Enable watching in development for faster iteration
6. **Set Priorities**: Understand source priority for proper overrides
7. **Use Defaults**: Provide sensible defaults for optional configuration
8. **Add Observers**: Use observers for dynamic configuration updates

## API Reference

See the [GoDoc](https://pkg.go.dev/github.com/os-golib/go-config) for complete API documentation.

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.