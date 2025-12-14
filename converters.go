package config

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// TypeConverter defines a function to convert a raw value to a target reflect.Value.
type TypeConverter func(dst reflect.Value, raw any) error

// TypeConverterRegistry manages type and kind converters.
type TypeConverterRegistry struct {
	kindConverters map[reflect.Kind]TypeConverter
	typeConverters map[reflect.Type]TypeConverter
}

// NewTypeConverterRegistry creates a new registry and registers default converters.
func NewTypeConverterRegistry() *TypeConverterRegistry {
	registry := &TypeConverterRegistry{
		kindConverters: make(map[reflect.Kind]TypeConverter),
		typeConverters: make(map[reflect.Type]TypeConverter),
	}

	registry.registerDefaults()
	return registry
}

// registerDefaults registers all built-in type converters.
func (r *TypeConverterRegistry) registerDefaults() {
	// Kind-based converters
	r.RegisterKindConverter(reflect.String, convertString)
	r.RegisterKindConverter(reflect.Bool, convertBool)
	r.RegisterKindConverter(reflect.Int, convertInt)
	r.RegisterKindConverter(reflect.Int8, convertInt)
	r.RegisterKindConverter(reflect.Int16, convertInt)
	r.RegisterKindConverter(reflect.Int32, convertInt)
	r.RegisterKindConverter(reflect.Int64, convertInt64)
	r.RegisterKindConverter(reflect.Uint, convertUint)
	r.RegisterKindConverter(reflect.Uint8, convertUint)
	r.RegisterKindConverter(reflect.Uint16, convertUint)
	r.RegisterKindConverter(reflect.Uint32, convertUint)
	r.RegisterKindConverter(reflect.Uint64, convertUint)
	r.RegisterKindConverter(reflect.Float32, convertFloat)
	r.RegisterKindConverter(reflect.Float64, convertFloat)
	r.RegisterKindConverter(reflect.Slice, convertSlice)
	r.RegisterKindConverter(reflect.Struct, convertStruct)

	// Type-specific converters (override kind-based)
	r.RegisterTypeConverter(reflect.TypeOf(time.Duration(0)), convertDuration)
	r.RegisterTypeConverter(reflect.TypeOf(url.URL{}), convertURL)
}

// RegisterKindConverter registers a converter for a reflect.Kind.
func (r *TypeConverterRegistry) RegisterKindConverter(kind reflect.Kind, converter TypeConverter) {
	r.kindConverters[kind] = converter
}

// RegisterTypeConverter registers a converter for a specific reflect.Type.
func (r *TypeConverterRegistry) RegisterTypeConverter(typ reflect.Type, converter TypeConverter) {
	r.typeConverters[typ] = converter
}

// Convert attempts to convert a raw value to the destination reflect.Value.
func (r *TypeConverterRegistry) Convert(dst reflect.Value, raw any) error {
	if !dst.CanSet() || raw == nil {
		return nil
	}

	dst = indirect(dst)

	// Direct assignment if types are compatible
	rv := reflect.ValueOf(raw)
	if rv.Type().AssignableTo(dst.Type()) {
		dst.Set(rv)
		return nil
	}

	// 1. Check for exact type converter first
	if conv, ok := r.typeConverters[dst.Type()]; ok {
		return conv(dst, raw)
	}

	// 2. Check for kind-based converter
	if conv, ok := r.kindConverters[dst.Kind()]; ok {
		return conv(dst, raw)
	}

	return fmt.Errorf("unsupported type conversion: from %T to %s", raw, dst.Type())
}

// --- Converter Implementations ---

func convertString(dst reflect.Value, raw any) error {
	dst.SetString(fmt.Sprint(raw))
	return nil
}

func convertBool(dst reflect.Value, raw any) error {
	b, err := strconv.ParseBool(fmt.Sprint(raw))
	if err != nil {
		return err
	}
	dst.SetBool(b)
	return nil
}

func convertInt(dst reflect.Value, raw any) error {
	i, err := strconv.ParseInt(fmt.Sprint(raw), 10, dst.Type().Bits())
	if err != nil {
		return err
	}
	dst.SetInt(i)
	return nil
}

func convertInt64(dst reflect.Value, raw any) error {
	// Special case for time.Duration
	if dst.Type() == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(fmt.Sprint(raw))
		if err != nil {
			return err
		}
		dst.SetInt(int64(d))
		return nil
	}
	return convertInt(dst, raw)
}

func convertUint(dst reflect.Value, raw any) error {
	u, err := strconv.ParseUint(fmt.Sprint(raw), 10, dst.Type().Bits())
	if err != nil {
		return err
	}
	dst.SetUint(u)
	return nil
}

func convertFloat(dst reflect.Value, raw any) error {
	f, err := strconv.ParseFloat(fmt.Sprint(raw), dst.Type().Bits())
	if err != nil {
		return err
	}
	dst.SetFloat(f)
	return nil
}

func convertSlice(dst reflect.Value, raw any) error {
	items := extractSliceItems(raw)
	slice := reflect.MakeSlice(dst.Type(), len(items), len(items))

	for i, item := range items {
		if err := assignValue(slice.Index(i), item); err != nil { // Recursively assign elements
			return err
		}
	}
	dst.Set(slice)
	return nil
}

func convertStruct(dst reflect.Value, raw any) error {
	// If the raw value is a map, we can attempt to bind it to the struct
	if m, ok := raw.(map[string]any); ok {
		// This requires a global or passed-in converter registry, creating a circular dependency.
		// To avoid this, we create a temporary binding function that doesn't rely on the main config.
		// A better approach in a real system might be dependency injection.
		// For now, we'll use a simplified binding that only handles direct field maps.
		return bindMapToStructSimple(m, dst.Addr().Interface())
	}
	return fmt.Errorf("cannot convert %T to struct", raw)
}

func convertDuration(dst reflect.Value, raw any) error {
	d, err := time.ParseDuration(fmt.Sprint(raw))
	if err != nil {
		return err
	}
	dst.SetInt(int64(d))
	return nil
}

func convertURL(dst reflect.Value, raw any) error {
	str := fmt.Sprint(raw)
	u, err := url.Parse(str)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	dst.Set(reflect.ValueOf(*u))
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// extractSliceItems converts various types to string slices.
func extractSliceItems(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		items := make([]string, len(v))
		for i, e := range v {
			items[i] = fmt.Sprint(e)
		}
		return items
	case string:
		// Support comma-separated values
		if strings.Contains(v, ",") {
			return strings.Split(v, ",")
		}
		return []string{v}
	default:
		return []string{fmt.Sprint(raw)}
	}
}

// assignValue is a recursive helper for convertSlice to handle nested types.
// It uses a temporary registry to avoid circular dependencies with the main Config.
// This is a simplified version; a full implementation would require passing the registry.
func assignValue(dst reflect.Value, raw any) error {
	// This is a simplified version for demonstration. A full implementation would
	// use the main TypeConverterRegistry. To avoid circular dependencies, we
	// create a temporary one here.
	tempRegistry := NewTypeConverterRegistry()
	return tempRegistry.Convert(dst, raw)
}

// bindMapToStructSimple is a minimal binder for convertStruct to avoid circular dependencies.
// It does not support nested structs or custom key transformations.
func bindMapToStructSimple(data map[string]any, dst any) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("destination must point to a struct")
	}

	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Field(i)
		fieldType := rt.Field(i)
		if !field.CanSet() || !fieldType.IsExported() {
			continue
		}

		name := strings.ToLower(fieldType.Name)
		if val, ok := data[name]; ok {
			if err := assignValue(field, val); err != nil {
				return fmt.Errorf("binding field %s: %w", name, err)
			}
		}
	}
	return nil
}
