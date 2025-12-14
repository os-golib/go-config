package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// =============================================================================
// Core Interfaces
// =============================================================================

// Source is a typed configuration provider.
// Higher priority overrides lower priority.
type Source interface {
	Name() string
	Priority() int
	Load() (map[string]any, error)
	WatchPaths() []string
}

// =============================================================================
// Base Source
// =============================================================================

type BaseSource struct {
	name     string
	priority int
	paths    []string
}

func NewBaseSource(name string, priority int, paths ...string) BaseSource {
	return BaseSource{name: name, priority: priority, paths: paths}
}

func (b BaseSource) Name() string         { return b.name }
func (b BaseSource) Priority() int        { return b.priority }
func (b BaseSource) WatchPaths() []string { return b.paths }

// =============================================================================
// Default Priorities (single source of truth)
// =============================================================================

const (
	DefaultMemoryPriority = 0
	DefaultFilePriority   = 10
	DefaultGlobPriority   = 10
	DefaultEnvPriority    = 20
)

// =============================================================================
// Source Creation
// =============================================================================

type SourceArgs struct {
	Path   string
	Data   map[string]any
	Prefix string
}

type SourceBuilder func(SourceArgs, int) Source

var sourceRegistry = map[string]SourceBuilder{
	"memory": func(a SourceArgs, p int) Source {
		return MemoryWithPriority(a.Data, p)
	},
	"file": func(a SourceArgs, p int) Source {
		return FileWithPriority(a.Path, p)
	},
	"glob": func(a SourceArgs, p int) Source {
		return GlobWithPriority(a.Path, p)
	},
	"env": func(a SourceArgs, p int) Source {
		return EnvWithPriority(a.Prefix, p)
	},
}

// CreateSource is the ONLY source factory entry point.
func CreateSource(kind string, args SourceArgs, priority int) Source {
	if builder, ok := sourceRegistry[kind]; ok {
		return builder(args, priority)
	}
	return autoDetectSource(args, priority)
}

func autoDetectSource(args SourceArgs, priority int) Source {
	if args.Path == "" {
		return MemoryWithPriority(args.Data, priority)
	}
	if isGlob(args.Path) {
		return GlobWithPriority(args.Path, priority)
	}
	return FileWithPriority(args.Path, priority)
}

func isGlob(path string) bool {
	for _, r := range path {
		switch r {
		case '*', '?', '[':
			return true
		}
	}
	return false
}

// =============================================================================
// Memory Source
// =============================================================================

type MemorySource struct {
	BaseSource
	data map[string]any
}

func Memory(data map[string]any) *MemorySource {
	return MemoryWithPriority(data, DefaultMemoryPriority)
}

func MemoryWithPriority(data map[string]any, priority int) *MemorySource {
	return &MemorySource{
		BaseSource: NewBaseSource("memory", priority),
		data:       cloneMap(data),
	}
}

func (s *MemorySource) Load() (map[string]any, error) {
	return cloneMap(s.data), nil
}

func (s *MemorySource) Update(data map[string]any) {
	s.data = cloneMap(data)
}

// =============================================================================
// File Source
// =============================================================================

type FileSource struct {
	BaseSource
	path    string
	decoder FileDecoder
}

func File(path string) *FileSource {
	return FileWithPriority(path, DefaultFilePriority)
}

func FileWithPriority(path string, priority int) *FileSource {
	return &FileSource{
		BaseSource: NewBaseSource("file:"+path, priority, path),
		path:       path,
		decoder:    decoderFor(path),
	}
}

func (s *FileSource) Load() (map[string]any, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var decoded map[string]any
	if err := s.decoder.Decode(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode file: %w", err)
	}

	return flattenToDot(decoded), nil
}

// =============================================================================
// File Decoders (strategy registry)
// =============================================================================

type FileDecoder interface {
	Decode([]byte, any) error
	Extensions() []string
}

type jsonDecoder struct{}
type yamlDecoder struct{}

func (jsonDecoder) Decode(b []byte, v any) error { return json.Unmarshal(b, v) }
func (jsonDecoder) Extensions() []string         { return []string{".json"} }

func (yamlDecoder) Decode(b []byte, v any) error { return yaml.Unmarshal(b, v) }
func (yamlDecoder) Extensions() []string {
	return []string{".yaml", ".yml"}
}

var decoders = []FileDecoder{
	jsonDecoder{},
	yamlDecoder{},
}

func RegisterDecoder(d FileDecoder) {
	decoders = append(decoders, d)
}

func decoderFor(path string) FileDecoder {
	ext := strings.ToLower(filepath.Ext(path))
	for _, d := range decoders {
		for _, e := range d.Extensions() {
			if e == ext {
				return d
			}
		}
	}
	return jsonDecoder{}
}

// =============================================================================
// Glob (Multi-File) Source
// =============================================================================

type MultiFileSource struct {
	BaseSource
	pattern string
}

func Glob(pattern string) *MultiFileSource {
	return GlobWithPriority(pattern, DefaultGlobPriority)
}

func GlobWithPriority(pattern string, priority int) *MultiFileSource {
	return &MultiFileSource{
		BaseSource: NewBaseSource("glob:"+pattern, priority),
		pattern:    pattern,
	}
}

func (s *MultiFileSource) Load() (map[string]any, error) {
	files, err := filepath.Glob(s.pattern)
	if err != nil {
		return nil, fmt.Errorf("glob pattern: %w", err)
	}

	out := make(map[string]any)
	for _, f := range files {
		data, err := FileWithPriority(f, s.Priority()).Load()
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", f, err)
		}
		for k, v := range data {
			out[k] = v
		}
	}
	return out, nil
}

// =============================================================================
// Environment Source
// =============================================================================

type KeyTransformer func(string) string

type EnvSource struct {
	BaseSource
	prefix    string
	transform KeyTransformer
}

func Environment(prefix string) *EnvSource {
	return EnvWithPriority(prefix, DefaultEnvPriority)
}

func EnvWithPriority(prefix string, priority int) *EnvSource {
	return &EnvSource{
		BaseSource: NewBaseSource("env", priority),
		prefix:     prefix,
		transform:  KeyTransforms.UnderscoreToDot,
	}
}

func (s *EnvSource) WithKeyTransform(fn KeyTransformer) *EnvSource {
	s.transform = fn
	return s
}

func (s *EnvSource) Load() (map[string]any, error) {
	out := make(map[string]any)

	for _, kv := range os.Environ() {
		k, v, ok := splitKeyValue(kv)
		if !ok {
			continue
		}

		if s.prefix != "" {
			if !strings.HasPrefix(k, s.prefix) {
				continue
			}
			k = strings.TrimPrefix(k, s.prefix)
		}

		if s.transform != nil {
			k = s.transform(k)
		}

		out[k] = v
	}
	return out, nil
}

// =============================================================================
// Flattening (single unified logic)
// =============================================================================

func flattenToDot(in map[string]any) map[string]any {
	out := make(map[string]any)
	flatten("", in, out)
	return out
}

// flatten recursively flattens nested structures.
func flatten(prefix string, v any, out map[string]any) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			flatten(joinKeys(prefix, k), val, out)
		}
	case map[any]any:
		m := make(map[string]any)
		for k, val := range x {
			m[fmt.Sprint(k)] = val
		}
		flatten(prefix, m, out)
	case []any:
		for i, val := range x {
			flatten(fmt.Sprintf("%s.%d", prefix, i), val, out)
		}
		out[prefix] = joinList(x)
	default:
		out[prefix] = x
	}
}

// =============================================================================
// Helpers
// =============================================================================

func splitKeyValue(s string) (key string, value string, ok bool) {
	i := strings.IndexByte(s, '=')
	if i <= 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

// joinKeys joins key parts with dots.
func joinKeys(a, b string) string {
	if a == "" {
		return b
	}
	return a + "." + b
}

func joinList(v []any) string {
	out := make([]string, len(v))
	for i, e := range v {
		out[i] = fmt.Sprint(e)
	}
	return strings.Join(out, ",")
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// KeyTransforms provides common key transformation functions.
var KeyTransforms = struct {
	Lower           KeyTransformer
	Upper           KeyTransformer
	DotToUnderscore KeyTransformer
	UnderscoreToDot KeyTransformer
	CamelToSnake    KeyTransformer
}{
	Lower: strings.ToLower,
	Upper: strings.ToUpper,
	DotToUnderscore: func(k string) string {
		return strings.ReplaceAll(k, ".", "_")
	},
	UnderscoreToDot: func(k string) string {
		return strings.ToLower(strings.ReplaceAll(k, "_", "."))
	},
	CamelToSnake: func(k string) string {
		var result strings.Builder
		for i, r := range k {
			if i > 0 && r >= 'A' && r <= 'Z' {
				result.WriteRune('_')
			}
			result.WriteRune(r)
		}
		return strings.ToLower(result.String())
	},
}
