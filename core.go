package config

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
)

// =============================================================================
// Core Configuration Manager
// =============================================================================

// Config is the central configuration manager with thread-safe operations.
type Config struct {
	mu              sync.RWMutex
	sources         []Source
	data            map[string]any
	validate        *validator.Validate
	validationRules map[string]string
	observers       []Observer
	ctx             context.Context
	cancel          context.CancelFunc

	// Extension points
	converter  *TypeConverterRegistry
	template   *TemplateProcessor
	encryption *EncryptionProcessor
	profiles   *ProfileManager
	hooks      *HookManager
}

// Observer receives notifications when configuration changes.
type Observer interface {
	OnConfigChange(changed map[string]any)
}

// ObserverFunc adapts a function to the Observer interface.
type ObserverFunc func(changed map[string]any)

func (f ObserverFunc) OnConfigChange(changed map[string]any) { f(changed) }

// New creates a configuration instance with sensible defaults.
func New(opts ...Option) *Config {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Config{
		data:            make(map[string]any),
		sources:         make([]Source, 0),
		validate:        validator.New(validator.WithRequiredStructEnabled()),
		validationRules: make(map[string]string),
		observers:       make([]Observer, 0),
		ctx:             ctx,
		cancel:          cancel,
		converter:       NewTypeConverterRegistry(),
		template:        NewTemplateProcessor(),
		hooks:           NewHookManager(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// =============================================================================
// Validation Rules Management
// =============================================================================

// AddRule adds a validation rule for a specific configuration key.
func (c *Config) AddRule(key string, rule string) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.validationRules[key] = rule
	return c
}

// AddRules adds multiple validation rules at once.
func (c *Config) AddRules(rules ...*validationRules) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, rule := range rules {
		c.validationRules[rule.Key()] = rule.String()
	}
	return c
}

// ValidateKey validates a specific key against its registered rules.
func (c *Config) ValidateKey(key string) error {
	c.mu.RLock()
	rule, exists := c.validationRules[key]
	value, hasValue := c.data[key]
	c.mu.RUnlock()

	if !exists {
		return nil // No rule registered
	}

	if !hasValue {
		if strings.Contains(rule, "required") {
			return fmt.Errorf("key %q is required but not found", key)
		}
		return nil
	}

	// Create a temporary struct to validate
	return c.validateValue(key, value, rule)
}

// ValidateAll validates all keys that have registered rules.
func (c *Config) ValidateAll() error {
	c.mu.RLock()
	rules := make(map[string]string, len(c.validationRules))
	for k, v := range c.validationRules {
		rules[k] = v
	}
	data := cloneMap(c.data)
	c.mu.RUnlock()

	errors := make(map[string]string)
	for key, rule := range rules {
		value, exists := data[key]
		if !exists {
			if strings.Contains(rule, "required") {
				errors[key] = "is required"
			}
			continue
		}

		if err := c.validateValue(key, value, rule); err != nil {
			errors[key] = err.Error()
		}
	}

	if len(errors) > 0 {
		return ValidationErrors{Errors: errors}
	}
	return nil
}

// validateValue validates a single value against a rule string.
func (c *Config) validateValue(_ string, value any, rule string) error {
	fieldName := "Value"
	structType := reflect.StructOf([]reflect.StructField{
		{
			Name: fieldName,
			Type: reflect.TypeOf(value),
			Tag:  reflect.StructTag(fmt.Sprintf(`validate:"%q"`, rule)),
		},
	})

	structValue := reflect.New(structType).Elem()
	structValue.Field(0).Set(reflect.ValueOf(value))

	if err := c.validate.Struct(structValue.Interface()); err != nil {
		if ve, ok := err.(validator.ValidationErrors); ok {
			for _, fe := range ve {
				return fmt.Errorf("%s", validationMessage(fe))
			}
		}
		return err
	}
	return nil
}

// =============================================================================
// Lifecycle Management
// =============================================================================

// Load loads all sources, merges data, and notifies observers of changes.
func (c *Config) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Pre-load hook
	if err := c.hooks.ExecutePreLoad(c); err != nil {
		return fmt.Errorf("pre-load hook: %w", err)
	}

	merged := make(map[string]any)

	for _, src := range c.sources {
		data, err := src.Load()
		if err != nil {
			return fmt.Errorf("source %s: %w", src.Name(), err)
		}
		deepMerge(merged, data)
	}

	// Post-load hook
	if err := c.hooks.ExecutePostLoad(c, merged); err != nil {
		return fmt.Errorf("post-load hook: %w", err)
	}

	changed := detectChanges(c.data, merged)
	c.data = merged

	if len(changed) > 0 {
		c.notifyObservers(changed)
	}

	c.mu.Unlock()
	if len(c.validationRules) > 0 {
		if err := c.ValidateAll(); err != nil {
			c.mu.Lock()
			return fmt.Errorf("validation failed: %w", err)
		}
	}
	c.mu.Lock()

	return nil
}

// Watch starts monitoring sources for changes and auto-reloads.
func (c *Config) Watch(interval time.Duration) error {
	paths := c.collectWatchPaths()
	if len(paths) == 0 {
		return fmt.Errorf("no watchable sources configured")
	}

	go c.watchLoop(interval, paths)
	return nil
}

// Close stops watching and releases resources.
func (c *Config) Close() error {
	c.cancel()
	return nil
}

// =============================================================================
// Source Management
// =============================================================================

// AddSource adds a configuration source with automatic sorting by priority.
func (c *Config) AddSource(src Source) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sources = append(c.sources, src)
	c.sortSources()
	return c
}

// AddSourceWithMiddleware adds a source with processing middleware.
func (c *Config) AddSourceWithMiddleware(src Source, middleware ...SourceMiddleware) *Config {
	for _, mw := range middleware {
		src = mw(src)
	}
	return c.AddSource(src)
}

// RemoveSource removes a source by name.
func (c *Config) RemoveSource(name string) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()

	filtered := make([]Source, 0, len(c.sources))
	for _, src := range c.sources {
		if src.Name() != name {
			filtered = append(filtered, src)
		}
	}
	c.sources = filtered
	return c
}

// =============================================================================
// Data Access
// =============================================================================

func GetEnv(key string) string {
	return os.Getenv(key)
}

// Get retrieves a value by key with type checking.
func (c *Config) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

// getTyped is a generic helper that reduces duplication in Get* methods.
func getTyped[T any](c *Config, key string, defaultVal []T, converter func(any) (T, bool)) T {
	if val, ok := c.Get(key); ok {
		if converted, ok := converter(val); ok {
			return converted
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	var zero T
	return zero
}

// GetString retrieves a string value with optional default.
func (c *Config) GetString(key string, defaultVal ...string) string {
	return getTyped(c, key, defaultVal, func(v any) (string, bool) {
		if s, ok := v.(string); ok {
			return s, true
		}
		return fmt.Sprint(v), true
	})
}

// GetInt retrieves an integer value with optional default.
func (c *Config) GetInt(key string, defaultVal ...int) int {
	return getTyped(c, key, defaultVal, func(v any) (int, bool) {
		if i, ok := v.(int); ok {
			return i, true
		}
		var result int
		_, err := fmt.Sscanf(fmt.Sprint(v), "%d", &result)
		return result, err == nil
	})
}

// GetBool retrieves a boolean value with optional default.
func (c *Config) GetBool(key string, defaultVal ...bool) bool {
	return getTyped(c, key, defaultVal, func(v any) (bool, bool) {
		if b, ok := v.(bool); ok {
			return b, true
		}
		s := fmt.Sprint(v)
		return s == "true" || s == "1" || s == "yes", true
	})
}

// GetDuration retrieves a duration value with optional default.
func (c *Config) GetDuration(key string, defaultVal ...time.Duration) time.Duration {
	return getTyped(c, key, defaultVal, func(v any) (time.Duration, bool) {
		if d, ok := v.(time.Duration); ok {
			return d, true
		}
		if s := fmt.Sprint(v); s != "" {
			if d, err := time.ParseDuration(s); err == nil {
				return d, true
			}
		}
		return 0, false
	})
}

// GetFloat retrieves a float64 value with optional default.
func (c *Config) GetFloat(key string, defaultVal ...float64) float64 {
	return getTyped(c, key, defaultVal, func(v any) (float64, bool) {
		if f, ok := v.(float64); ok {
			return f, true
		}
		var result float64
		_, err := fmt.Sscanf(fmt.Sprint(v), "%f", &result)
		return result, err == nil
	})
}

// GetStringSlice retrieves a string slice value with optional default.
func (c *Config) GetStringSlice(key string, defaultVal ...[]string) []string {
	return getTyped(c, key, defaultVal, func(v any) ([]string, bool) {
		switch val := v.(type) {
		case []string:
			return val, true
		case string:
			return strings.Split(val, ","), true
		case []any:
			result := make([]string, len(val))
			for i, item := range val {
				result[i] = fmt.Sprint(item)
			}
			return result, true
		}
		return nil, false
	})
}

// MustGet panics if the key doesn't exist.
func (c *Config) MustGet(key string) any {
	val, ok := c.Get(key)
	if !ok {
		panic(fmt.Sprintf("required config key %q not found", key))
	}
	return val
}

// Set updates a configuration value at runtime (memory source).
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

// AllKeys returns all configuration keys.
func (c *Config) AllKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// =============================================================================
// Binding & Validation
// =============================================================================

// Bind binds configuration data to a struct.
func (c *Config) Bind(dst any) error {
	c.mu.RLock()
	data := cloneMap(c.data)
	c.mu.RUnlock()

	return c.bindMapToStruct(data, dst)
}

func (c *Config) BindAndValidate(dst any) error {
	if err := c.Bind(dst); err != nil {
		return err
	}
	return c.Validate(dst)
}

// BindWithRules binds data and validates against registered rules.
func (c *Config) BindWithRules(dst any) error {
	if err := c.Bind(dst); err != nil {
		return err
	}
	if err := c.ValidateAll(); err != nil {
		return err
	}
	return c.Validate(dst)
}

// Validate validates a struct using the configured validator.
func (c *Config) Validate(dst any) error {
	if err := c.validate.Struct(dst); err != nil {
		return wrapValidationError(err)
	}
	return nil
}

// =============================================================================
// Observation
// =============================================================================

// Observe registers an observer for configuration changes.
func (c *Config) Observe(obs Observer) *Config {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.observers = append(c.observers, obs)
	return c
}

// ObserveFunc registers a function as an observer.
func (c *Config) ObserveFunc(fn func(changed map[string]any)) *Config {
	return c.Observe(ObserverFunc(fn))
}

// =============================================================================
// Extension Management
// =============================================================================

// EnableProfiles activates profile management.
func (c *Config) EnableProfiles() *ProfileManager {
	if c.profiles == nil {
		c.profiles = NewProfileManager(c)
	}
	return c.profiles
}

// SetEncryptionProcessor configures encryption support.
func (c *Config) SetEncryptionProcessor(processor *EncryptionProcessor) {
	c.encryption = processor
}

// RegisterTypeConverter registers a custom type converter.
func (c *Config) RegisterTypeConverter(kind reflect.Kind, converter TypeConverter) {
	c.converter.RegisterKindConverter(kind, converter)
}

// RegisterHook registers lifecycle hooks.
func (c *Config) RegisterHook(hook Hook) {
	c.hooks.Register(hook)
}

// AddTemplateFunction adds a custom template function.
func (c *Config) AddTemplateFunction(name string, fn interface{}) {
	c.template.AddFunction(name, fn)
}

// =============================================================================
// Internal Helpers
// =============================================================================

func (c *Config) sortSources() {
	// Insertion sort - optimal for small lists
	for i := 1; i < len(c.sources); i++ {
		cur := c.sources[i]
		j := i - 1
		for j >= 0 && c.sources[j].Priority() > cur.Priority() {
			c.sources[j+1] = c.sources[j]
			j--
		}
		c.sources[j+1] = cur
	}
}

func (c *Config) notifyObservers(changed map[string]any) {
	for _, obs := range c.observers {
		go obs.OnConfigChange(cloneMap(changed))
	}
}

func (c *Config) collectWatchPaths() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var paths []string
	for _, src := range c.sources {
		paths = append(paths, src.WatchPaths()...)
	}
	return paths
}

func (c *Config) watchLoop(interval time.Duration, paths []string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	modTimes := make(map[string]time.Time)
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			modTimes[path] = info.ModTime()
		}
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if c.hasChanges(modTimes) {
				_ = c.Load() // Errors logged via hooks
			}
		}
	}
}

func (c *Config) hasChanges(modTimes map[string]time.Time) bool {
	for path, oldTime := range modTimes {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().After(oldTime) {
			modTimes[path] = info.ModTime()
			return true
		}
	}
	return false
}

func (c *Config) bindMapToStruct(data map[string]any, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("destination must point to a struct")
	}

	for key, val := range data {
		path := splitPath(key)
		if err := c.setByPath(rv, path, val); err != nil {
			return fmt.Errorf("bind %q: %w", key, err)
		}
	}

	return nil
}

func (c *Config) setByPath(v reflect.Value, path []string, raw any) error {
	if len(path) == 0 {
		return nil
	}

	v = indirect(v)

	if v.Kind() != reflect.Struct {
		return nil
	}

	field, ok := findField(v, path[0])
	if !ok {
		return fmt.Errorf("unknown config field %q on %s", path[0], v.Type())
	}

	if len(path) == 1 {
		return c.converter.Convert(field, raw)
	}

	return c.setByPath(field, path[1:], raw)
}

// =============================================================================
// Options Pattern
// =============================================================================

type Option func(*Config)

func WithContext(ctx context.Context) Option {
	return func(c *Config) {
		c.ctx, c.cancel = context.WithCancel(ctx)
	}
}

func WithValidator(v *validator.Validate) Option {
	return func(c *Config) {
		c.validate = v
	}
}

//
// =============================================================================
// Validation Errors
// =============================================================================
//

type ValidationErrors struct {
	Errors map[string]string
}

func (e ValidationErrors) Error() string {
	parts := make([]string, 0, len(e.Errors))
	for field, msg := range e.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	return "configuration validation failed: " + strings.Join(parts, "; ")
}

func wrapValidationError(err error) error {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	out := make(map[string]string, len(ve))
	for _, fe := range ve {
		key := strings.ToLower(fe.Namespace())
		out[key] = validationMessage(fe)
	}

	return ValidationErrors{Errors: out}
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be >= " + fe.Param()
	case "max":
		return "must be <= " + fe.Param()
	case "email":
		return "must be a valid email"
	case "url":
		return "must be a valid URL"
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		return fmt.Sprintf("validation failed: %s", fe.Tag())
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func deepMerge(dst, src map[string]any) {
	for k, v := range src {
		if dstVal, exists := dst[k]; exists {
			if dstMap, dstOk := dstVal.(map[string]any); dstOk {
				if srcMap, srcOk := v.(map[string]any); srcOk {
					deepMerge(dstMap, srcMap)
					continue
				}
			}
		}
		dst[k] = v
	}
}

func detectChanges(old, updated map[string]any) map[string]any {
	changed := make(map[string]any)
	for k, newVal := range updated {
		if oldVal, exists := old[k]; !exists || !deepEqual(oldVal, newVal) {
			changed[k] = newVal
		}
	}
	return changed
}

func deepEqual(a, b any) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func splitPath(key string) []string {
	return strings.Split(key, ".")
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func findField(v reflect.Value, name string) (reflect.Value, bool) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		if matchField(sf, name) {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

// matchField checks if a struct field matches a key name.
func matchField(sf reflect.StructField, key string) bool {
	// 1. Check config tag
	if tag := sf.Tag.Get("config"); tag != "" {
		return strings.EqualFold(tag, key)
	}
	// 2. Check json tag
	if tag := sf.Tag.Get("json"); tag != "" {
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			return strings.EqualFold(parts[0], key)
		}
	}
	// 3. Fallback to field name
	return strings.EqualFold(sf.Name, key)
}
