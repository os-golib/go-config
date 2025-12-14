package config

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"
)

// TemplateProcessor processes configuration values using Go templates.
type TemplateProcessor struct {
	funcMap template.FuncMap
}

// NewTemplateProcessor creates a new TemplateProcessor with default functions.
func NewTemplateProcessor() *TemplateProcessor {
	return &TemplateProcessor{
		funcMap: template.FuncMap{
			"env":        os.Getenv,
			"lower":      strings.ToLower,
			"upper":      strings.ToUpper,
			"split":      strings.Split,
			"join":       strings.Join,
			"replace":    strings.ReplaceAll,
			"contains":   strings.Contains,
			"hasPrefix":  strings.HasPrefix,
			"hasSuffix":  strings.HasSuffix,
			"trim":       strings.TrimSpace,
			"trimSpace":  strings.TrimSpace,
			"trimPrefix": strings.TrimPrefix,
			"trimSuffix": strings.TrimSuffix,
			"repeat":     strings.Repeat,
			"toUpper":    strings.ToUpper,
			"toLower":    strings.ToLower,
			// "title":      strings.Title,
			"eq":         reflect.DeepEqual,
			"ne":         reflect.DeepEqual,
			"lt":         func(a, b int) bool { return a < b },
			"le":         func(a, b int) bool { return a <= b },
			"gt":         func(a, b int) bool { return a > b },
			"ge":         func(a, b int) bool { return a >= b },
			"and":        func(a, b bool) bool { return a && b },
			"or":         func(a, b bool) bool { return a || b },
			"not":        func(a bool) bool { return !a },
			"type":       func(v any) string { return reflect.TypeOf(v).String() },
			"len":        func(v any) int { return reflect.ValueOf(v).Len() },
			"format":     fmt.Sprintf,
			"formatBool": func(b bool) string { return fmt.Sprintf("%t", b) },
			"formatUint": func(u uint64) string { return fmt.Sprintf("%d", u) },
			"formatInt":  func(i int) string { return fmt.Sprintf("%d", i) },
			"formatFloat": func(f float64, precision int) string {
				return fmt.Sprintf(fmt.Sprintf("%%.%df", precision), f)
			},
			"default": func(def, val string) string {
				if val == "" {
					return def
				}
				return val
			},
		},
	}
}

// AddFunction adds a custom function to the template processor's function map.
func (tp *TemplateProcessor) AddFunction(name string, fn interface{}) {
	tp.funcMap[name] = fn
}

// Process recursively processes a configuration map, executing any templates found in string values.
func (tp *TemplateProcessor) Process(data map[string]any) (map[string]any, error) {
	result := make(map[string]any)
	for key, value := range data {
		processed, err := tp.processValue(value, data)
		if err != nil {
			return nil, fmt.Errorf("processing key %q: %w", key, err)
		}
		result[key] = processed
	}
	return result, nil
}

// processValue recursively processes a value, handling maps, slices, and strings.
func (tp *TemplateProcessor) processValue(value any, ctx map[string]any) (any, error) {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "{{") && strings.Contains(v, "}}") {
			tmpl, err := template.New("config").
				Funcs(tp.funcMap).
				Option("missingkey=error").
				Parse(v)
			if err != nil {
				return nil, err
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, ctx); err != nil {
				return nil, err
			}
			return buf.String(), nil
		}
		return v, nil

	case map[string]any:
		out := make(map[string]any)
		for k, val := range v {
			p, err := tp.processValue(val, ctx)
			if err != nil {
				return nil, err
			}
			out[k] = p
		}
		return out, nil

	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			p, err := tp.processValue(val, ctx)
			if err != nil {
				return nil, err
			}
			out[i] = p
		}
		return out, nil

	default:
		return v, nil
	}
}

// TemplateSource is a wrapper that applies template processing to another source.
type TemplateSource struct {
	BaseSource
	source    Source
	processor *TemplateProcessor
}

// NewTemplateSource creates a new TemplateSource.
func NewTemplateSource(source Source, processor *TemplateProcessor) *TemplateSource {
	return &TemplateSource{
		BaseSource: NewBaseSource("template:"+source.Name(), source.Priority()),
		source:     source,
		processor:  processor,
	}
}

// Load loads data from the underlying source and processes it with templates.
func (s *TemplateSource) Load() (map[string]any, error) {
	data, err := s.source.Load()
	if err != nil {
		return nil, err
	}
	return s.processor.Process(data)
}

// WatchPaths returns the watch paths from the underlying source.
func (s *TemplateSource) WatchPaths() []string {
	return s.source.WatchPaths()
}
