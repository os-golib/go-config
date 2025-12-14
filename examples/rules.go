package main

// Example 2: Validate on Load

import (
	"fmt"
	"log"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/os-golib/go-config"
)

// =============================================================================
// Example 3: Complex Validation Rules
// =============================================================================

func Example3_ComplexValidation() {
	cfg := config.NewBuilder().
		AddFile("config.yaml").
		AddEnv("APP_").
		AddRules(
			// Server rules
			config.Rules.Required("server.host"),
			config.Rules.Range("server.port", 1024, 65535),
			config.Rules.Required("server.read_timeout"),
			config.Rules.Required("server.write_timeout"),
			config.Rules.Range("server.max_connections", 1, 10000),

			// Database rules
			config.Rules.Required("database.host"),
			config.Rules.Range("database.port", 1024, 65535),
			config.Rules.Required("database.username"),
			config.Rules.Min("database.username", 3),
			config.Rules.Required("database.password"),
			config.Rules.Min("database.password", 8),
			config.Rules.Required("database.name"),

			// App rules
			config.Rules.Required("environment"),
			config.Rules.OneOf("environment", "dev", "staging", "prod"),
			config.Rules.Required("admin.email"),
			config.Rules.Email("admin.email"),
			config.Rules.Required("api.endpoint"),
			config.Rules.URL("api.endpoint"),
		).
		Build()

	// Bind to structs
	var serverCfg ServerConfig
	if err := cfg.BindAndValidate(&serverCfg); err != nil {
		log.Fatal(err)
	}

	var dbCfg Database
	if err := cfg.BindAndValidate(&dbCfg); err != nil {
		log.Fatal(err)
	}

	var appCfg AppConfig
	if err := cfg.BindAndValidate(&appCfg); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Server: %s:%d\n", serverCfg.Host, serverCfg.Port)
	fmt.Printf("Database: %s:%d/%s\n", dbCfg.Host, dbCfg.Port, dbCfg.Name)
	fmt.Printf("Environment: %s\n", appCfg.App.Environment)
}

// =============================================================================
// Example 4: Custom Validation Rules
// =============================================================================

func Example4_CustomRules() {
	cfg := config.NewBuilder().
		AddFile("config.yaml").
		// Use V10 for custom validator tags
		AddRules(
			config.Rules.V10("api.key", "required,alphanum,len=32"),
			config.Rules.V10("redis.url", "required,redis"),
			config.Rules.V10("kafka.brokers", "required,dive,hostname_port"),
		).
		// Register custom validators
		RegisterValidation("redis", func(fl validator.FieldLevel) bool {
			val := fl.Field().String()
			return len(val) > 0 && (val[:6] == "redis:" || val[:8] == "rediss:")
		}).
		Build()

	if err := cfg.ValidateAll(); err != nil {
		log.Fatal(err)
	}
}

// =============================================================================
// Example 5: Conditional Validation
// =============================================================================

func Example5_ConditionalValidation() {
	cfg := config.NewBuilder().
		AddFile("config.yaml").
		AddEnv("APP_").
		Build()

	env := cfg.GetString("environment")

	// Add different rules based on environment
	if env == "prod" {
		cfg.AddRules(
			config.Rules.Required("ssl.cert"),
			config.Rules.Required("ssl.key"),
			config.Rules.Required("monitoring.endpoint"),
			config.Rules.Min("workers", 4),
		)
	} else {
		cfg.AddRules(
			config.Rules.Min("workers", 1),
		)
	}

	// Validate with environment-specific rules
	if err := cfg.ValidateAll(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Running in %s mode\n", env)
}

// =============================================================================
// Example 6: Validate Individual Keys
// =============================================================================

func Example6_ValidateIndividualKeys() {
	cfg := config.NewBuilder().
		AddMemory(map[string]any{
			"server.port": 8080,
			"admin.email": "admin@example.com",
		}).
		AddRules(
			config.Rules.Range("server.port", 1024, 65535),
			config.Rules.Email("admin.email"),
		).
		Build()

	// Validate specific keys
	if err := cfg.ValidateKey("server.port"); err != nil {
		log.Printf("Invalid port: %v", err)
	}

	if err := cfg.ValidateKey("admin.email"); err != nil {
		log.Printf("Invalid email: %v", err)
	}

	// Update and revalidate
	cfg.Set("server.port", 99999)
	if err := cfg.ValidateKey("server.port"); err != nil {
		log.Printf("Port out of range: %v", err)
		cfg.Set("server.port", 8080) // Reset to valid value
	}
}

// =============================================================================
// Example 7: Chained Rules
// =============================================================================

func Example7_ChainedRules() {
	cfg := config.NewBuilder().
		AddFile("config.yaml").
		AddRules(
			// Chain multiple validations for a single key
			config.Rules.Required("api.token").
				Add("alphanum", "").
				Add("min", "16").
				Add("max", "64"),

			config.Rules.Required("cache.ttl").
				Add("min", "1s").
				Add("max", "24h"),

			config.Rules.Email("notifications.email").
				Add("required", ""),
		).
		Build()

	if err := cfg.ValidateAll(); err != nil {
		log.Fatal(err)
	}
}

// =============================================================================
// Example 8: Production-Ready Setup
// =============================================================================

func Example8_ProductionSetup() {
	cfg := config.NewBuilder().
		WithDefaultPriority(10).
		AddFile("/etc/myapp/config.yaml").
		AddFile("/etc/myapp/secrets.yaml").
		AddEnv("MYAPP_").
		WithCaching(5*time.Minute).
		WithRetry(3, time.Second).
		// Comprehensive validation rules
		AddRules(
			// Server configuration
			config.Rules.Required("server.host"),
			config.Rules.Required("server.port"),
			config.Rules.Range("server.port", 1024, 65535),
			config.Rules.Min("server.workers", 1),
			config.Rules.Max("server.workers", 1000),
			config.Rules.Required("server.read_timeout"),
			config.Rules.Required("server.write_timeout"),

			// Database configuration
			config.Rules.Required("database.host"),
			config.Rules.Required("database.port"),
			config.Rules.Range("database.port", 1024, 65535),
			config.Rules.Required("database.name"),
			config.Rules.Required("database.username"),
			config.Rules.Required("database.password"),
			config.Rules.Min("database.password", 12),
			config.Rules.Min("database.max_connections", 1),
			config.Rules.Max("database.max_connections", 1000),

			// Security configuration
			config.Rules.Required("security.jwt_secret"),
			config.Rules.Min("security.jwt_secret", 32),
			config.Rules.Required("security.allowed_origins"),
			config.Rules.OneOf("security.tls_version", "1.2", "1.3"),

			// Monitoring configuration
			config.Rules.URL("monitoring.metrics_endpoint"),
			config.Rules.URL("monitoring.tracing_endpoint"),
			config.Rules.Email("monitoring.alert_email"),
		).
		AddLoggingHook(myLogger).
		MustBuild()

	fmt.Println("Configuration loaded successfully.", cfg)

	fmt.Println("Production configuration loaded and validated!")
}

// Mock logger for example
var myLogger = &mockLogger{}

type mockLogger struct{}

func (l *mockLogger) Info(msg string, args ...any) {
	fmt.Printf("INFO: %s %v\n", msg, args)
}

func (l *mockLogger) Error(msg string, args ...any) {
	fmt.Printf("ERROR: %s %v\n", msg, args)
}
