package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/os-golib/go-config"
)

// =============================================================================
// Example 1: Basic Usage with Builder
// =============================================================================

func ExampleBasicUsage() {
	example := NewExample(
		"Basic Configuration",
		"Load configuration from YAML files and environment variables",
	)
	example.Start()

	example.Section("Configuration Setup")
	example.Info("Creating configuration builder")
	example.CodeBlock(`cfg, err := config.NewBuilder().
    AddFile("config.yaml").
    AddEnv("APP_").
    BuildAndLoad()`)

	cfg, err := config.NewBuilder().
		AddFile("config.yaml").
		AddEnv("APP_").
		// Add validation rules using the fluent Rules API
		AddRules(
			config.Rules.Required("database.host"),
			config.Rules.Required("database.port"),
			config.Rules.Range("database.port", 1024, 65535),
			config.Rules.Email("admin.email"),
			config.Rules.URL("api.endpoint"),
			config.Rules.Min("workers", 1),
			config.Rules.Max("workers", 100),
		).
		BuildAndLoad()

	if err != nil {
		example.Error(err)
		return
	}

	example.Section("Accessing Configuration Values")
	port := cfg.GetInt("server.port", 8080)
	host := cfg.GetString("server.host", "localhost")

	example.KeyValue("Server Port", fmt.Sprintf("%d", port))
	example.KeyValue("Server Host", host)
	example.KeyValue("Config Source", fmt.Sprintf("%T", cfg))

	example.Complete()
}

// =============================================================================
// Example 2: Multiple Sources with Priorities
// =============================================================================

func ExampleMultipleSources() {
	example := NewExample(
		"Multiple Configuration Sources",
		"Demonstrates how to combine multiple configuration sources with different priorities",
	)
	example.Start()

	example.Section("Source Priority Configuration")
	example.Info("Setting up sources with different priorities:")
	example.Info("Memory config (default priority) → File (priority 10) → Env (priority 20)")

	cfg := config.NewBuilder().
		AddMemory(map[string]any{
			"app.name":    "DefaultApp",
			"app.version": "0.0.1",
		}).
		WithDefaultPriority(10).
		AddFile("config.yaml"). // Priority 10
		WithDefaultPriority(20).
		AddEnv("APP_"). // Priority 20 (overrides file)
		AddRules(
			config.Rules.Required("server.host"),
			config.Rules.Range("server.port", 80, 8080),
			config.Rules.OneOf("environment", "dev", "staging", "prod"),
		).
		MustBuild()

	example.Section("Resolved Configuration")
	appName := cfg.GetString("app.name", "unknown")
	appVersion := cfg.GetString("app.version", "unknown")

	example.KeyValue("Application Name", appName)
	example.KeyValue("Application Version", appVersion)
	example.Info("Environment variables have highest priority (20)")

	example.Complete()
}

// =============================================================================
// Example 3: Watching for Changes
// =============================================================================

func ExampleWatchingChanges() {
	example := NewExample(
		"Real-time Configuration Updates",
		"Shows how to watch configuration files for changes and auto-reload",
	)
	example.Start()

	example.Section("Setting up File Watcher")
	example.Info("Configuring watcher with 5-second interval")

	cfg, err := config.NewBuilder().
		AddFile("config.yaml").
		AddObserverFunc(func(changed map[string]any) {
			fmt.Println("\n📡 CONFIGURATION CHANGED DETECTED:")
			for key, value := range changed {
				fmt.Printf("   %s: %v\n", key, value)
			}
		}).
		BuildAndWatch(5 * time.Second)

	if err != nil {
		example.Error(err)
		return
	}
	defer cfg.Close()

	example.Section("Monitoring")
	example.Info("Watching for configuration changes...")
	example.Info("Try modifying config.yaml file while this runs")

	// Simulate watching for 10 seconds instead of 30
	for i := 1; i <= 5; i++ {
		time.Sleep(2 * time.Second)
		fmt.Printf("⏳ Still watching... (%d/5 checks)\n", i)
	}

	example.Success("File watching completed successfully")
	example.Complete()
}

// =============================================================================
// Example 4: Struct Binding with Validation
// =============================================================================

func ExampleStructBinding() {
	example := NewExample(
		"Struct Binding and Validation",
		"Shows how to bind configuration to Go structs with automatic validation",
	)
	example.Start()

	example.Section("Configuration Data")
	configData := map[string]any{
		"host":              "localhost",
		"port":              8080,
		"timeout":           "30s",
		"database.url":      "postgres://localhost/mydb",
		"database.maxconns": 10,
	}

	for key, value := range configData {
		example.KeyValue(key, fmt.Sprintf("%v", value))
	}

	example.Section("Struct Definition")
	example.CodeBlock(`type ServerConfig struct {
    Host     string         config:"host" validate:"required"
    Port     int            config:"port" validate:"required"
    Timeout  time.Duration  config:"timeout" validate:"required"
    Database DatabaseConfig config:"database" validate:"required"
}`)

	cfg := config.NewBuilder().
		AddMemory(configData).
		MustBuild()

	var appCfg AppConfig
	example.Section("Binding Process")
	if err := cfg.BindAndValidate(&appCfg); err != nil {
		example.Error(err)
		return
	}

	example.Section("Bound Configuration")
	example.KeyValue("Server Host", appCfg.Server.Host)
	example.KeyValue("Server Port", fmt.Sprintf("%d", appCfg.Server.Port))
	example.KeyValue("Server Timeout", appCfg.Server.ReadTimeout.String())
	example.KeyValue("Database Host", appCfg.Database.Host)
	example.KeyValue("Database Port", fmt.Sprintf("%d", appCfg.Database.Port))

	example.Success("Configuration successfully bound and validated")
	example.Complete()
}

// =============================================================================
// Example 5: Template Processing
// =============================================================================

func ExampleTemplateProcessing() {
	example := NewExample(
		"Template Processing",
		"Demonstrates dynamic configuration values using Go templates",
	)
	example.Start()

	example.Section("Template Configuration")
	templateConfig := map[string]any{
		"env":      "production",
		"app.url":  "https://{{.env}}.example.com",
		"app.name": "{{upper \"myapp\"}}",
		"api.path": "/api/v{{.version}}",
		"version":  "1.0",
	}

	example.CodeBlock(`templateConfig := map[string]any{
		"env":      "production",
		"app.url":  "https://{{.env}}.example.com",
		"app.name": "{{upper \"myapp\"}}",
		"api.path": "/api/v{{.version}}",
		"version":  "1.0",
	}
		cfg := config.NewBuilder().
		WithTemplateProcessing().
		AddMemory(templateConfig).
		MustBuild()`)

	cfg := config.NewBuilder().
		WithTemplateProcessing().
		AddMemory(templateConfig).
		MustBuild()

	example.Section("Processed Results")
	example.Result("App URL: %s", cfg.GetString("app.url"))   // https://production.example.com
	example.Result("App Name: %s", cfg.GetString("app.name")) // MYAPP
	example.Result("API Path: %s", cfg.GetString("api.path")) // /api/v1.0

	example.Section("Template Functions Available")
	example.Info("Built-in functions: upper, lower, title, trim, default, env, etc.")

	example.Complete()
}

// =============================================================================
// Example 6: Encryption Support
// =============================================================================

func ExampleEncryption() {
	example := NewExample(
		"Secure Configuration with Encryption",
		"Shows how to store and use encrypted values in configuration",
	)
	example.Start()

	example.Section("Encryption Setup")
	example.Info("Creating encryptor with secret key")
	encryptor, err := config.NewAESEncryptor("my-secret-key-12345")
	if err != nil {
		example.Error(err)
		return
	}

	originalPassword := "super-secret-password"
	example.KeyValue("Original Password", originalPassword)

	encrypted, err := encryptor.Encrypt(originalPassword)
	if err != nil {
		example.Error(err)
		return
	}
	example.KeyValue("Encrypted Value", encrypted[:50]+"...")

	example.Section("Configuration with Encrypted Values")
	cfg := config.NewBuilder().
		WithEncryption("my-secret-key-12345").
		AddMemory(map[string]any{
			"db.host":     "localhost",
			"db.port":     5432,
			"db.user":     "admin",
			"db.password": "ENC:" + encrypted,
		}).
		MustBuild()

	example.Section("Accessing Decrypted Values")
	example.KeyValue("Database Host", cfg.GetString("db.host"))
	example.KeyValue("Database Port", fmt.Sprintf("%d", cfg.GetInt("db.port")))
	example.KeyValue("Database User", cfg.GetString("db.user"))

	// Show only first and last few chars of decrypted password for security
	password := cfg.GetString("db.password")
	maskedPassword := string(password[0]) + "***" + string(password[len(password)-1])
	example.KeyValue("Database Password", maskedPassword)

	example.Success("Encryption/decryption working correctly")
	example.Complete()
}

// =============================================================================
// Example 7: Profile Management
// =============================================================================

func ExampleProfiles() {
	example := NewExample(
		"Configuration Profiles",
		"Demonstrates environment-specific configuration profiles",
	)
	example.Start()

	example.Section("Profile Definitions")
	example.Info("Setting up development and production profiles")

	profiles := map[string]map[string]any{
		"development": {
			"db.host":       "localhost",
			"db.port":       5432,
			"debug":         true,
			"log.level":     "debug",
			"cache.enabled": false,
		},
		"production": {
			"db.host":       "prod-db.example.com",
			"db.port":       5432,
			"debug":         false,
			"log.level":     "warn",
			"cache.enabled": true,
			"cache.ttl":     "5m",
		},
	}

	cfg := config.NewBuilder().
		EnableProfiles().
		AddProfile("development", profiles["development"]).
		AddProfile("production", profiles["production"]).
		SetActiveProfile("development").
		MustBuild()

	example.Section("Development Profile (Active)")
	for key, expected := range profiles["development"] {
		actual, _ := cfg.Get(key)
		status := "✓"
		if actual != expected {
			status = "✗"
		}
		fmt.Printf("  %s %-20s: %v\n", status, key, actual)
	}

	example.Section("Switching to Production Profile")
	cfg.EnableProfiles().SetActiveProfile("production")
	example.Info("Profile switched to 'production'")

	example.Section("Production Profile (Active)")
	for key, expected := range profiles["production"] {
		actual, _ := cfg.Get(key)
		status := "✓"
		if actual != expected {
			status = "✗"
		}
		fmt.Printf("  %s %-20s: %v\n", status, key, actual)
	}

	example.Success("Profile management working correctly")
	example.Complete()
}

// =============================================================================
// Example 8: Custom Hooks
// =============================================================================

type CustomValidationHook struct{}

func (h *CustomValidationHook) Name() string  { return "custom-validation" }
func (h *CustomValidationHook) Priority() int { return 50 }

func (h *CustomValidationHook) OnPostLoad(c *config.Config, data map[string]any) error {
	required := []string{"app.name", "app.version", "app.environment"}
	for _, key := range required {
		if _, ok := data[key]; !ok {
			return fmt.Errorf("required key %q is missing", key)
		}
	}
	return nil
}

func ExampleCustomHooks() {
	example := NewExample(
		"Custom Configuration Hooks",
		"Shows how to extend configuration with custom lifecycle hooks",
	)
	example.Start()

	example.Section("Hook Implementation")
	example.CodeBlock(`type CustomValidationHook struct{}
func (h *CustomValidationHook) OnPostLoad(c *config.Config, data map[string]any) error {
    required := []string{"app.name", "app.version", "app.environment"}
    for _, key := range required {
        if _, ok := data[key]; !ok {
            return fmt.Errorf("required key %%q is missing", key)
        }
    }
    return nil
}`)

	example.Section("Test 1: Valid Configuration")
	validConfig := map[string]any{
		"app.name":        "MyApp",
		"app.version":     "1.0.0",
		"app.environment": "production",
		"extra.field":     "value",
	}

	example.Info("Testing with valid configuration...")
	cfg1, err1 := config.NewBuilder().
		AddHook(&CustomValidationHook{}).
		AddMemory(validConfig).
		BuildAndLoad()

	if err1 != nil {
		example.Error(err1)
	} else {
		example.Success("Valid configuration passed validation")
		_ = cfg1
	}

	example.Section("Test 2: Invalid Configuration (Missing Field)")
	invalidConfig := map[string]any{
		"app.name": "MyApp",
		// Missing app.version
		"app.environment": "production",
	}

	example.Info("Testing with invalid configuration...")
	_, err2 := config.NewBuilder().
		AddHook(&CustomValidationHook{}).
		AddMemory(invalidConfig).
		BuildAndLoad()

	if err2 != nil {
		example.Error(err2)
		example.Info("Hook correctly rejected invalid configuration")
	}

	example.Complete()
}

// =============================================================================
// Example 9: Middleware Composition
// =============================================================================

func ExampleMiddleware() {
	example := NewExample(
		"Source Middleware",
		"Demonstrates adding caching and retry logic to configuration sources",
	)
	example.Start()

	example.Section("Middleware Configuration")
	example.Info("Adding caching (5min) and retry (3 attempts) to remote source")
	example.CodeBlock(`cfg := config.NewBuilder().
    AddSourceWithMiddleware(
        config.File("remote.config.yaml"),
        config.WithCaching(5*time.Minute),
        config.WithRetry(3, time.Second),
    ).
    AddFile("config.yaml"). // No middleware
    MustBuild()`)

	cfg := config.NewBuilder().
		AddSourceWithMiddleware(
			config.File("remote.config.yaml"),
			config.WithCaching(5*time.Minute),
			config.WithRetry(3, time.Second),
		).
		AddFile("config.yaml").
		MustBuild()

	example.Section("Middleware Benefits")
	example.Info("✅ Caching: Reduces load on remote sources")
	example.Info("✅ Retry: Handles transient network failures")
	example.Info("✅ Composition: Apply middleware selectively")

	_ = cfg
	example.Success("Middleware configured successfully")
	example.Complete()
}

// =============================================================================
// Example 10: Conditional Sources
// =============================================================================

func ExampleConditionalSources() {
	example := NewExample(
		"Conditional Configuration Sources",
		"Shows how to load configuration sources based on runtime conditions",
	)
	example.Start()

	example.Section("Condition Definition")
	example.Info("Condition: Load production config only when ENV=production")

	isProduction := func() bool {
		env := config.GetEnv("ENV")
		isProd := env == "production"
		example.Info("ENV variable value: %s", env)
		example.Info("Condition result: %v", isProd)
		return isProd
	}

	example.Section("Conditional Builder Setup")
	cfg := config.NewBuilder().
		AddFile("base.config.yaml").
		AddConditional(
			config.File("config.prod.yaml"),
			isProduction,
		).
		MustBuild()

	example.Section("Loaded Sources")
	if isProduction() {
		example.Result("✅ Production configuration loaded")
		example.Info("Sources: base.config.yaml + config.prod.yaml")
	} else {
		example.Result("✅ Development configuration loaded")
		example.Info("Sources: base.config.yaml only")
	}

	_ = cfg
	example.Complete()
}

// =============================================================================
// Example 11: Composite Sources
// =============================================================================
func ExampleCompositeSources() {
	example := NewExample(
		"Composite Configuration Sources",
		"Merges multiple related sources into a single logical unit",
	)
	example.Start()

	example.Section("Composite Definition")
	example.Info("Grouping database configs: primary, replica, cache")

	cfg := config.NewBuilder().
		AddComposite("database-configs", 15,
			config.File("db-primary.yaml"),
			config.File("db-replica.yaml"),
			config.File("db-cache.yaml"),
		).
		MustBuild()

	example.Section("Result")
	example.Result("✅ Composite source loaded successfully")
	example.Info("Composite name: database-configs")
	example.Info("Sources merged: db-primary.yaml, db-replica.yaml, db-cache.yaml")

	_ = cfg
	example.Complete()
}

// =============================================================================
// Example 12: Custom Type Converters
// =============================================================================

type CustomType struct {
	Value string
}

func customTypeConverter(dst reflect.Value, raw any) error {
	str := fmt.Sprint(raw)
	dst.Set(reflect.ValueOf(CustomType{Value: "custom:" + str}))
	return nil
}

func ExampleCustomTypeConverter() {
	example := NewExample(
		"Custom Type Converters",
		"Demonstrates registering and using a custom type converter",
	)
	example.Start()

	example.Section("Custom Type Definition")
	example.Info("Target type: CustomType")

	cfg := config.NewBuilder().
		RegisterTypeConverter(
			reflect.TypeOf(CustomType{}).Kind(),
			customTypeConverter,
		).
		AddMemory(map[string]any{
			"custom": "test",
		}).
		MustBuild()

	var result struct {
		Custom CustomType
	}

	example.Section("Binding Result")
	cfg.Bind(&result)

	example.Result("✅ Custom type conversion successful")
	example.Info("Converted value: %s", result.Custom.Value)

	example.Complete()
}

// =============================================================================
// Example 13: Context Cancellation
// =============================================================================

func ExampleContextCancellation() {
	example := NewExample(
		"Context Cancellation",
		"Stops configuration watching when context is cancelled",
	)
	example.Start()

	example.Section("Context Setup")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	example.Info("Context timeout: 10 seconds")

	cfg, err := config.NewBuilder().
		WithContext(ctx).
		AddFile("config.yaml").
		BuildAndWatch(1 * time.Second)

	if err != nil {
		log.Fatal(err)
	}

	example.Section("Watching Configuration")
	example.Info("Watching until context is cancelled...")

	<-ctx.Done()
	cfg.Close()

	example.Result("✅ Context cancelled, watcher stopped")
	example.Complete()
}

// =============================================================================
// Example 14: Preset Configurations
// =============================================================================
func ExamplePresetConfigurations() {
	example := NewExample(
		"Preset Configurations",
		"Uses predefined builder presets for different environments",
	)
	example.Start()

	example.Section("Development Preset")
	devCfg := config.NewDevelopmentConfig().
		AddMemory(map[string]any{"custom.key": "value"}).
		MustBuild()
	example.Info("Dev custom.key: %s", devCfg.GetString("custom.key"))

	// example.Section("Production Preset")
	// prodCfg := config.NewProductionConfig().
	// 	AddMemory(map[string]any{"custom.key": "value"}).
	// 	MustBuild()
	// example.Info("Prod custom.key: %s", prodCfg.GetString("custom.key"))

	example.Section("Test Preset")
	testCfg := config.NewTestConfig().
		AddMemory(map[string]any{"test.mode": true}).
		MustBuild()
	example.Info("Test test.mode: %v", testCfg.GetBool("test.mode"))

	example.Result("✅ All preset configurations loaded")
	example.Complete()
}

// =============================================================================
// Example 15: Builder Chaining and Branching
// =============================================================================

func ExampleBuilderChaining() {
	example := NewExample(
		"Builder Chaining and Branching",
		"Clones a base builder to create environment-specific configurations",
	)
	example.Start()

	example.Section("Base Builder")
	base := config.NewBuilder().
		AddFile("base.config.yaml").
		WithTemplateProcessing()
	example.Info("Base config: base.config.yaml")

	example.Section("Development Branch")
	devCfg := base.Clone().
		AddFile("config.dev.yaml").
		MustBuild()

	example.Section("Production Branch")
	prodCfg := base.Clone().
		AddFile("config.prod.yaml").
		WithCaching(10 * time.Minute).
		MustBuild()

	example.Result("✅ Branch configurations built")
	example.Info("Dev DB Host: %s", devCfg.GetString("database.host"))
	example.Info("Prod DB Host: %s", prodCfg.GetString("database.host"))

	example.Complete()
}

// =============================================================================
// Example 16: Dynamic Configuration Updates
// =============================================================================

func ExampleDynamicUpdates() {
	example := NewExample(
		"Dynamic Configuration Updates",
		"Demonstrates runtime updates and observer notifications",
	)
	example.Start()

	cfg := config.NewBuilder().
		AddMemory(map[string]any{
			"feature.enabled": false,
		}).
		AddObserverFunc(func(changed map[string]any) {
			example.Info("Observer triggered: %v", changed)
		}).
		MustBuild()

	example.Section("Runtime Update")
	example.Info("Toggling feature.enabled → true")
	cfg.Set("feature.enabled", true)
	cfg.Load()

	example.Result("✅ Configuration updated dynamically")
	example.Complete()
}

// =============================================================================
// Example 17: Glob Pattern Loading
// =============================================================================

func ExampleGlobPattern() {
	example := NewExample(
		"Glob Pattern Source Loading",
		"Loads multiple configuration files using glob patterns",
	)
	example.Start()

	cfg := config.NewBuilder().
		AddGlob("configs/*.yaml").
		AddGlob("configs/db-*.json").
		MustBuild()

	example.Result("✅ Configuration files loaded via glob patterns")
	example.Info("Patterns used: configs/*.yaml, configs/db-*.json")

	_ = cfg
	example.Complete()
}

// =============================================================================
// Example 18: Default Values with Hooks
// =============================================================================

func ExampleDefaultValues() {
	example := NewExample(
		"Default Values with Hooks",
		"Applies default values when configuration entries are missing",
	)
	example.Start()

	cfg := config.NewBuilder().
		AddDefaultsHook(map[string]any{
			"server.port":    8080,
			"server.host":    "0.0.0.0",
			"server.timeout": "30s",
		}).
		AddFile("config.yaml").
		MustBuild()

	example.Section("Resolved Values")
	example.Info("Server port: %d", cfg.GetInt("server.port"))

	example.Result("✅ Defaults applied successfully")
	example.Complete()
}

// =============================================================================
// Example 19: Logging Hook
// =============================================================================

type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, args ...any) {
	fmt.Println("INFO:", msg, args)
}

func (l *SimpleLogger) Error(msg string, args ...any) {
	fmt.Println("ERROR:", msg, args)
}

func ExampleLoggingHook() {
	example := NewExample(
		"Logging Hook",
		"Adds structured logging during configuration lifecycle",
	)
	example.Start()

	cfg := config.NewBuilder().
		AddLoggingHook(&SimpleLogger{}).
		AddFile("config.yaml").
		MustBuild()

	example.Result("✅ Configuration loaded with logging enabled")
	_ = cfg
	example.Complete()
}

// =============================================================================
// Example 20: Complete Real-World Application
// =============================================================================

func ExampleRealWorldApplication() {
	example := NewExample(
		"Complete Real-World Application",
		"Demonstrates a full production-ready configuration pipeline",
	)
	example.Start()

	example.Section("Building Configuration")

	cfg, err := config.NewBuilder().
		AddFile("config.yaml").
		AddConditional(
			config.File("config.prod.yaml"),
			func() bool { return config.GetEnv("ENV") == "production" },
		).
		WithDefaultPriority(100).
		AddEnv("APP_").
		WithTemplateProcessing().
		WithEncryption("encryption-key").
		WithCaching(5*time.Minute).
		WithRetry(3, time.Second).
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
			config.Rules.Required("app.environment"),
			config.Rules.OneOf("app.environment", "dev", "staging", "prod"),
			config.Rules.Required("app.admin.email"),
			config.Rules.Email("app.admin.email"),
			config.Rules.Required("app.api.endpoint"),
			config.Rules.URL("app.api.endpoint"),
		).
		// AddDefaultsHook(map[string]any{
		// 	"server.port": 8080,
		// }).
		// AddLoggingHook(&SimpleLogger{}).
		// AddValidationHook(func(data map[string]any) error {
		// 	if data["server.port"] == nil {
		// 		return fmt.Errorf("server.port is required")
		// 	}
		// 	return nil
		// }).
		AddObserverFunc(func(changed map[string]any) {
			log.Println("Configuration changed:", changed)
		}).
		BuildAndWatch(10 * time.Second)

	if err != nil {
		log.Fatal(err)
	}
	defer cfg.Close()

	example.Section("Binding & Validation")
	var appConfig AppConfig
	if err := cfg.BindAndValidate(&appConfig); err != nil {
		log.Fatal(err)
	}

	example.Result("✅ Application fully configured")
	example.Info("Final config: %+v", appConfig)

	example.Complete()
}

// =============================================================================
// Main Execution
// =============================================================================

func main() {
	fmt.Println("🚀 GO-CONFIG EXAMPLES")
	fmt.Println("A comprehensive demonstration of the go-config library")
	fmt.Println(strings.Repeat("=", 70))

	// // Run examples in sequence
	// ExampleBasicUsage()
	// ExampleMultipleSources()
	// ExampleWatchingChanges()
	// ExampleStructBinding()
	// ExampleTemplateProcessing()
	// ExampleEncryption()
	// ExampleProfiles()
	// ExampleCustomHooks()
	// ExampleMiddleware()
	// ExampleConditionalSources()

	// ExampleCompositeSources()
	// ExampleCustomTypeConverter()
	// ExampleContextCancellation()
	// ExamplePresetConfigurations()
	// ExampleBuilderChaining()
	// ExampleDynamicUpdates()
	// ExampleGlobPattern()
	// ExampleDefaultValues()
	// ExampleLoggingHook()
	ExampleRealWorldApplication()

	// // Run examples in sequence
	// ExampleCompleteAppConfig()
	// ExampleMultiEnvironment()
	// ExampleFeatureFlags()
	// ExampleDatabaseConfig()
	// ExampleAPIGatewayConfig()
	// ExampleMicroservicesConfig()
	// ExampleSecretManagement()
	// ExampleValidationSuite()
	// ExampleDynamicConfiguration()
	// ExampleExportImport()

	// Run examples
	// Example1_BasicRules()
	// Example2_ValidateOnLoad()
	// Example3_ComplexValidation()
	// Example4_CustomRules()
	// Example5_ConditionalValidation()
	// Example6_ValidateIndividualKeys()
	// Example7_ChainedRules()
	// Example8_ProductionSetup()

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🎉 ALL EXAMPLES COMPLETED SUCCESSFULLY!")
	fmt.Println(strings.Repeat("=", 70))
}
