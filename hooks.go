package config

import (
	"fmt"
	"time"
)

// =============================================================================
// Hook System
// =============================================================================

// Hook defines lifecycle extension points for configuration operations.
type Hook interface {
	Name() string
	Priority() int // Lower executes first
}

// PreLoadHook executes before loading configuration.
type PreLoadHook interface {
	Hook
	OnPreLoad(c *Config) error
}

// PostLoadHook executes after loading configuration.
type PostLoadHook interface {
	Hook
	OnPostLoad(c *Config, data map[string]any) error
}

// PreBindHook executes before binding data to a struct.
type PreBindHook interface {
	Hook
	OnPreBind(c *Config, dst any) error
}

// PostBindHook executes after binding data to a struct.
type PostBindHook interface {
	Hook
	OnPostBind(c *Config, dst any) error
}

// HookManager orchestrates hook execution.
type HookManager struct {
	preLoad  []PreLoadHook
	postLoad []PostLoadHook
	preBind  []PreBindHook
	postBind []PostBindHook
}

// NewHookManager creates a new hook manager.
func NewHookManager() *HookManager {
	return &HookManager{
		preLoad:  make([]PreLoadHook, 0),
		postLoad: make([]PostLoadHook, 0),
		preBind:  make([]PreBindHook, 0),
		postBind: make([]PostBindHook, 0),
	}
}

// Register registers a hook (auto-detects type).
func (hm *HookManager) Register(hook Hook) {
	if h, ok := hook.(PreLoadHook); ok {
		hm.preLoad = append(hm.preLoad, h)
		sortHooks(hm.preLoad)
	}
	if h, ok := hook.(PostLoadHook); ok {
		hm.postLoad = append(hm.postLoad, h)
		sortHooks(hm.postLoad)
	}
	if h, ok := hook.(PreBindHook); ok {
		hm.preBind = append(hm.preBind, h)
		sortHooks(hm.preBind)
	}
	if h, ok := hook.(PostBindHook); ok {
		hm.postBind = append(hm.postBind, h)
		sortHooks(hm.postBind)
	}
}

// ExecutePreLoad executes all pre-load hooks.
func (hm *HookManager) ExecutePreLoad(c *Config) error {
	for _, hook := range hm.preLoad {
		if err := hook.OnPreLoad(c); err != nil {
			return fmt.Errorf("pre-load hook %s: %w", hook.Name(), err)
		}
	}
	return nil
}

// ExecutePostLoad executes all post-load hooks.
func (hm *HookManager) ExecutePostLoad(c *Config, data map[string]any) error {
	for _, hook := range hm.postLoad {
		if err := hook.OnPostLoad(c, data); err != nil {
			return fmt.Errorf("post-load hook %s: %w", hook.Name(), err)
		}
	}
	return nil
}

// ExecutePreBind executes all pre-bind hooks.
func (hm *HookManager) ExecutePreBind(c *Config, dst any) error {
	for _, hook := range hm.preBind {
		if err := hook.OnPreBind(c, dst); err != nil {
			return fmt.Errorf("pre-bind hook %s: %w", hook.Name(), err)
		}
	}
	return nil
}

// ExecutePostBind executes all post-bind hooks.
func (hm *HookManager) ExecutePostBind(c *Config, dst any) error {
	for _, hook := range hm.postBind {
		if err := hook.OnPostBind(c, dst); err != nil {
			return fmt.Errorf("post-bind hook %s: %w", hook.Name(), err)
		}
	}
	return nil
}

// sortHooks is a generic hook sorter using interface constraints.
func sortHooks[T Hook](hooks []T) {
	for i := 1; i < len(hooks); i++ {
		cur := hooks[i]
		j := i - 1
		for j >= 0 && hooks[j].Priority() > cur.Priority() {
			hooks[j+1] = hooks[j]
			j--
		}
		hooks[j+1] = cur
	}
}

// =============================================================================
// Built-in Hooks
// =============================================================================

// LoggingHook logs configuration operations.
type LoggingHook struct {
	logger Logger
}

type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

func NewLoggingHook(logger Logger) *LoggingHook {
	return &LoggingHook{logger: logger}
}

func (h *LoggingHook) Name() string  { return "logging" }
func (h *LoggingHook) Priority() int { return 1000 } // Low priority (runs last)

func (h *LoggingHook) OnPreLoad(c *Config) error {
	h.logger.Info("Loading configuration", "sources", len(c.sources))
	return nil
}

func (h *LoggingHook) OnPostLoad(_ *Config, data map[string]any) error {
	h.logger.Info("Configuration loaded", "keys", len(data))
	return nil
}

// ValidationHook validates configuration after loading.
type ValidationHook struct {
	validator func(data map[string]any) error
}

func NewValidationHook(validator func(data map[string]any) error) *ValidationHook {
	return &ValidationHook{validator: validator}
}

func (h *ValidationHook) Name() string  { return "validation" }
func (h *ValidationHook) Priority() int { return 50 }

func (h *ValidationHook) OnPostLoad(_ *Config, data map[string]any) error {
	return h.validator(data)
}

// DefaultsHook applies default values for missing keys.
type DefaultsHook struct {
	defaults map[string]any
}

func NewDefaultsHook(defaults map[string]any) *DefaultsHook {
	return &DefaultsHook{defaults: defaults}
}

func (h *DefaultsHook) Name() string  { return "defaults" }
func (h *DefaultsHook) Priority() int { return 10 } // Early execution

func (h *DefaultsHook) OnPostLoad(_ *Config, data map[string]any) error {
	for key, defaultVal := range h.defaults {
		if _, exists := data[key]; !exists {
			data[key] = defaultVal
		}
	}
	return nil
}

// =============================================================================
// Source Middleware
// =============================================================================

// SourceMiddleware wraps a source with additional behavior.
type SourceMiddleware func(Source) Source

// WithTemplate wraps a source with template processing.
func WithTemplate(processor *TemplateProcessor) SourceMiddleware {
	return func(src Source) Source {
		return NewTemplateSource(src, processor)
	}
}

// WithEncryption wraps a source with encryption support.
func WithEncryption(processor *EncryptionProcessor) SourceMiddleware {
	return func(src Source) Source {
		return NewEncryptionSource(src, processor)
	}
}

// WithCaching wraps a source with caching.
func WithCaching(ttl time.Duration) SourceMiddleware {
	return func(src Source) Source {
		return NewCachedSource(src, ttl)
	}
}

// WithRetry wraps a source with retry logic.
func WithRetry(maxAttempts int, backoff time.Duration) SourceMiddleware {
	return func(src Source) Source {
		return NewRetrySource(src, maxAttempts, backoff)
	}
}

// ChainMiddleware chains multiple middleware functions.
func ChainMiddleware(middleware ...SourceMiddleware) SourceMiddleware {
	return func(src Source) Source {
		for i := len(middleware) - 1; i >= 0; i-- {
			src = middleware[i](src)
		}
		return src
	}
}

// =============================================================================
// Middleware Implementations
// =============================================================================

// CachedSource caches the result of a source for a specified duration.
type CachedSource struct {
	BaseSource
	source   Source
	cache    map[string]any
	cachedAt time.Time
	ttl      time.Duration
}

func NewCachedSource(source Source, ttl time.Duration) *CachedSource {
	return &CachedSource{
		BaseSource: NewBaseSource("cached:"+source.Name(), source.Priority()),
		source:     source,
		cache:      nil,
		ttl:        ttl,
	}
}

func (s *CachedSource) Load() (map[string]any, error) {
	if s.cache != nil && time.Since(s.cachedAt) < s.ttl {
		return cloneMap(s.cache), nil
	}

	data, err := s.source.Load()
	if err != nil {
		return nil, err
	}

	s.cache = cloneMap(data)
	s.cachedAt = time.Now()
	return data, nil
}

func (s *CachedSource) WatchPaths() []string {
	return s.source.WatchPaths()
}

// RetrySource retries failed loads with exponential backoff.
type RetrySource struct {
	BaseSource
	source      Source
	maxAttempts int
	backoff     time.Duration
}

func NewRetrySource(source Source, maxAttempts int, backoff time.Duration) *RetrySource {
	return &RetrySource{
		BaseSource:  NewBaseSource("retry:"+source.Name(), source.Priority()),
		source:      source,
		maxAttempts: maxAttempts,
		backoff:     backoff,
	}
}

func (s *RetrySource) Load() (map[string]any, error) {
	var lastErr error
	for attempt := 0; attempt < s.maxAttempts; attempt++ {
		data, err := s.source.Load()
		if err == nil {
			return data, nil
		}
		lastErr = err
		if attempt < s.maxAttempts-1 {
			time.Sleep(s.backoff * time.Duration(attempt+1))
		}
	}
	return nil, fmt.Errorf("failed after %d attempts: %w", s.maxAttempts, lastErr)
}

func (s *RetrySource) WatchPaths() []string {
	return s.source.WatchPaths()
}

// =============================================================================
// Composite Source
// =============================================================================

// CompositeSource merges multiple sources as a single logical source.
type CompositeSource struct {
	BaseSource
	sources []Source
}

func NewCompositeSource(name string, priority int, sources ...Source) *CompositeSource {
	return &CompositeSource{
		BaseSource: NewBaseSource(name, priority),
		sources:    sources,
	}
}

func (s *CompositeSource) Load() (map[string]any, error) {
	merged := make(map[string]any)
	for _, src := range s.sources {
		data, err := src.Load()
		if err != nil {
			return nil, fmt.Errorf("composite source %s: %w", src.Name(), err)
		}
		deepMerge(merged, data)
	}
	return merged, nil
}

func (s *CompositeSource) WatchPaths() []string {
	var paths []string
	for _, src := range s.sources {
		paths = append(paths, src.WatchPaths()...)
	}
	return paths
}

// AddSource adds a source to the composite.
func (s *CompositeSource) AddSource(src Source) {
	s.sources = append(s.sources, src)
}

// =============================================================================
// Conditional Source
// =============================================================================

// ConditionalSource loads data conditionally based on a predicate.
type ConditionalSource struct {
	BaseSource
	source    Source
	condition func() bool
}

func NewConditionalSource(source Source, condition func() bool) *ConditionalSource {
	return &ConditionalSource{
		BaseSource: NewBaseSource("conditional:"+source.Name(), source.Priority()),
		source:     source,
		condition:  condition,
	}
}

func (s *ConditionalSource) Load() (map[string]any, error) {
	if !s.condition() {
		return make(map[string]any), nil
	}
	return s.source.Load()
}

func (s *ConditionalSource) WatchPaths() []string {
	if s.condition() {
		return s.source.WatchPaths()
	}
	return nil
}
