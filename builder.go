package config

import (
	"context"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
)

// =============================================================================
// API Fluent Builder
// =============================================================================

// Builder provides a fluent interface for building configurations.
type Builder struct {
	config     *Config
	factory    *SourceFactory
	middleware []SourceMiddleware
}

// NewBuilder creates a new builder with sensible defaults.
func NewBuilder() *Builder {
	return &Builder{
		config:     New(),
		factory:    NewSourceFactory(0),
		middleware: make([]SourceMiddleware, 0),
	}
}

// =============================================================================
// Context & Core Configuration
// =============================================================================

// WithContext sets the context for the configuration.
func (b *Builder) WithContext(ctx context.Context) *Builder {
	b.config.ctx, b.config.cancel = context.WithCancel(ctx)
	return b
}

// WithValidator sets a custom validator.
func (b *Builder) WithValidator(v *validator.Validate) *Builder {
	b.config.validate = v
	return b
}

// WithDefaultPriority sets the default priority for subsequently added sources.
func (b *Builder) WithDefaultPriority(priority int) *Builder {
	b.factory = NewSourceFactory(priority)
	return b
}

// =============================================================================
// Middleware Configuration
// =============================================================================

// WithMiddleware adds middleware to be applied to all sources.
func (b *Builder) WithMiddleware(mw ...SourceMiddleware) *Builder {
	b.middleware = append(b.middleware, mw...)
	return b
}

// WithTemplateProcessing enables template processing for all sources.
func (b *Builder) WithTemplateProcessing() *Builder {
	b.middleware = append(b.middleware, WithTemplate(b.config.template))
	return b
}

// WithEncryption enables encryption for all sources.
func (b *Builder) WithEncryption(key string) *Builder {
	encryptor, err := NewAESEncryptor(key)
	if err != nil {
		panic(err) // In builder, panic is acceptable for config errors
	}
	processor := NewEncryptionProcessor(encryptor, "ENC:")
	b.config.SetEncryptionProcessor(processor)
	b.middleware = append(b.middleware, WithEncryption(processor))
	return b
}

// WithCaching enables caching for all sources.
func (b *Builder) WithCaching(ttl time.Duration) *Builder {
	b.middleware = append(b.middleware, WithCaching(ttl))
	return b
}

// WithRetry enables retry logic for all sources.
func (b *Builder) WithRetry(attempts int, backoff time.Duration) *Builder {
	b.middleware = append(b.middleware, WithRetry(attempts, backoff))
	return b
}

// =============================================================================
// Source Management - Generic Add Method
// =============================================================================

// AddSource adds a generic source with middleware applied.
func (b *Builder) AddSource(src Source) *Builder {
	if len(b.middleware) > 0 {
		src = ChainMiddleware(b.middleware...)(src)
	}
	b.config.AddSource(src)
	return b
}

// AddSourceWithMiddleware adds a source with specific middleware.
func (b *Builder) AddSourceWithMiddleware(src Source, mw ...SourceMiddleware) *Builder {
	src = ChainMiddleware(mw...)(src)
	b.config.AddSource(src)
	return b
}

// =============================================================================
// Convenience Methods - Factory-Based Sources
// =============================================================================

// AddMemory adds a memory source.
func (b *Builder) AddMemory(data map[string]any) *Builder {
	return b.AddSource(b.factory.CreateMemorySource(data))
}

// AddFile adds a file source.
func (b *Builder) AddFile(path string) *Builder {
	return b.AddSource(b.factory.CreateFileSource(path))
}

// AddEnv adds an environment variable source.
func (b *Builder) AddEnv(prefix string) *Builder {
	return b.AddSource(b.factory.CreateEnvSource(prefix))
}

// AddGlob adds a multi-file source using glob patterns.
func (b *Builder) AddGlob(pattern string) *Builder {
	return b.AddSource(b.factory.CreateMultiFileSource(pattern))
}

// AddFiles adds multiple file sources at once.
func (b *Builder) AddFiles(paths ...string) *Builder {
	for _, path := range paths {
		b.AddFile(path)
	}
	return b
}

// =============================================================================
// Advanced Sources
// =============================================================================

// AddComposite adds a composite source merging multiple sources.
func (b *Builder) AddComposite(name string, priority int, sources ...Source) *Builder {
	return b.AddSource(NewCompositeSource(name, priority, sources...))
}

// AddConditional adds a conditional source.
func (b *Builder) AddConditional(src Source, condition func() bool) *Builder {
	return b.AddSource(NewConditionalSource(src, condition))
}

// =============================================================================
// Observation
// =============================================================================

// AddObserver adds an observer for configuration changes.
func (b *Builder) AddObserver(observer Observer) *Builder {
	b.config.Observe(observer)
	return b
}

// AddObserverFunc adds a function observer.
func (b *Builder) AddObserverFunc(fn func(changed map[string]any)) *Builder {
	b.config.ObserveFunc(fn)
	return b
}

// =============================================================================
// Hooks
// =============================================================================

// AddHook registers a lifecycle hook.
func (b *Builder) AddHook(hook Hook) *Builder {
	b.config.RegisterHook(hook)
	return b
}

// AddLoggingHook adds a logging hook.
func (b *Builder) AddLoggingHook(logger Logger) *Builder {
	return b.AddHook(NewLoggingHook(logger))
}

// AddValidationHook adds a validation hook.
func (b *Builder) AddValidationHook(validator func(map[string]any) error) *Builder {
	return b.AddHook(NewValidationHook(validator))
}

// AddDefaultsHook adds a defaults hook.
func (b *Builder) AddDefaultsHook(defaults map[string]any) *Builder {
	return b.AddHook(NewDefaultsHook(defaults))
}

// =============================================================================
// Extensions
// =============================================================================

// EnableProfiles enables profile management.
func (b *Builder) EnableProfiles() *Builder {
	b.config.EnableProfiles()
	return b
}

// AddProfile adds a configuration profile (requires EnableProfiles).
func (b *Builder) AddProfile(name string, data map[string]any) *Builder {
	pm := b.config.EnableProfiles()
	pm.AddProfile(name, data)
	return b
}

// SetActiveProfile sets the active profile (requires EnableProfiles).
func (b *Builder) SetActiveProfile(name string) *Builder {
	pm := b.config.EnableProfiles()
	if err := pm.SetActiveProfile(name); err != nil {
		panic(err)
	}
	return b
}

// AddTemplateFunction adds a custom template function.
func (b *Builder) AddTemplateFunction(name string, fn interface{}) *Builder {
	b.config.AddTemplateFunction(name, fn)
	return b
}

// =============================================================================
// Type Converters
// =============================================================================

// RegisterTypeConverter registers a custom type converter.
func (b *Builder) RegisterTypeConverter(kind reflect.Kind, converter TypeConverter) *Builder {
	b.config.RegisterTypeConverter(kind, converter)
	return b
}

// RegisterValidation registers a custom validation rule.
func (b *Builder) RegisterValidation(tag string, fn validator.Func) *Builder {
	if err := b.config.validate.RegisterValidation(tag, fn); err != nil {
		panic(err)
	}
	return b
}

// =============================================================================
// AddRules support
// =============================================================================

// AddRule adds a validation rule for a configuration key.
func (b *Builder) AddRule(key string, rule string) *Builder {
	b.config.AddRule(key, rule)
	return b
}

// AddRules adds multiple validation rules at once.
func (b *Builder) AddRules(rules ...*validationRules) *Builder {
	b.config.AddRules(rules...)
	return b
}

// =============================================================================
// Build Methods
// =============================================================================

// Build creates the final configuration instance without loading.
func (b *Builder) Build() *Config {
	return b.config
}

// MustBuild builds and loads, panicking on error.
func (b *Builder) MustBuild() *Config {
	if err := b.config.Load(); err != nil {
		panic(err)
	}
	return b.config
}

// BuildAndLoad loads the configuration and returns the instance.
func (b *Builder) BuildAndLoad() (*Config, error) {
	if err := b.config.Load(); err != nil {
		return nil, err
	}
	return b.config, nil
}

// BuildAndWatch loads and starts watching for changes.
func (b *Builder) BuildAndWatch(interval time.Duration) (*Config, error) {
	if err := b.config.Load(); err != nil {
		return nil, err
	}
	if err := b.config.Watch(interval); err != nil {
		return nil, err
	}
	return b.config, nil
}

// MustBuildAndWatch builds, loads, and watches, panicking on error.
func (b *Builder) MustBuildAndWatch(interval time.Duration) *Config {
	config, err := b.BuildAndWatch(interval)
	if err != nil {
		panic(err)
	}
	return config
}

// NewDevelopmentConfig creates a builder with development-friendly defaults.
func NewDevelopmentConfig() *Builder {
	return NewBuilder().
		WithDefaultPriority(10).
		AddFile("config.dev.yaml").
		AddEnv("DEV_").
		WithTemplateProcessing()
}

// NewProductionConfig creates a builder with production-ready defaults.
func NewProductionConfig() *Builder {
	return NewBuilder().
		WithDefaultPriority(10).
		AddFile("/etc/app/config.yaml").
		AddEnv("APP_").
		WithCaching(5*time.Minute).
		WithRetry(3, time.Second)
}

// NewTestConfig creates a builder for testing.
func NewTestConfig() *Builder {
	return NewBuilder().
		AddMemory(map[string]any{
			"env": "test",
		})
}

// Apply applies a configuration function to the builder.
func (b *Builder) Apply(fn func(*Builder) *Builder) *Builder {
	return fn(b)
}

// ApplyIf conditionally applies a configuration function.
func (b *Builder) ApplyIf(condition bool, fn func(*Builder) *Builder) *Builder {
	if condition {
		return fn(b)
	}
	return b
}

// Clone creates a copy of the builder for branching configuration.
func (b *Builder) Clone() *Builder {
	return &Builder{
		config:     b.config, // Shared config
		factory:    NewSourceFactory(b.factory.defaultPriority),
		middleware: append([]SourceMiddleware{}, b.middleware...),
	}
}
