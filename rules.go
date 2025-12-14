package config

import (
	"fmt"
	"strings"
)

// =============================================================================
// Validator Tag Constants (go-playground/validator v10 compatible)
// =============================================================================

const (
	TagRequired = "required"

	TagMin = "min"
	TagMax = "max"

	TagGT  = "gt"
	TagLT  = "lt"
	TagGTE = "gte"
	TagLTE = "lte"

	TagEQ = "eq"
	TagNE = "ne"

	TagEmail = "email"
	TagURL   = "url"
	TagUUID  = "uuid"
	TagUUID4 = "uuid4"

	TagLen    = "len"
	TagOneOf  = "oneof"
	TagRegexp = "regexp"
)

// =============================================================================
// Fluent Validation Rules
// =============================================================================

// validationRules represents a chainable set of validator rules for a key.
type validationRules struct {
	key  string
	tags []string
}

func newValidationRules(key string) *validationRules {
	return &validationRules{key: key}
}

// Add appends a validator tag (optionally with a parameter).
func (v *validationRules) Add(tag, param string) *validationRules {
	if param != "" {
		v.tags = append(v.tags, tag+"="+param)
	} else {
		v.tags = append(v.tags, tag)
	}
	return v
}

// String converts the rule set into a validator-compatible tag string.
func (v *validationRules) String() string {
	return strings.Join(v.tags, ",")
}

// Key returns the config key the rules apply to.
func (v *validationRules) Key() string {
	return v.key
}

// =============================================================================
// Rules Factory Methods
// =============================================================================

var Rules = struct {
	Required func(key string) *validationRules
	Range    func(key string, min, max int) *validationRules
	Min      func(key string, min int) *validationRules
	Max      func(key string, max int) *validationRules
	Email    func(key string) *validationRules
	URL      func(key string) *validationRules
	UUID     func(key string, version ...int) *validationRules
	Len      func(key string, length int) *validationRules
	OneOf    func(key string, values ...string) *validationRules
	Pattern  func(key, pattern string) *validationRules
	Gt       func(key string, value any) *validationRules
	Lt       func(key string, value any) *validationRules
	Gte      func(key string, value any) *validationRules
	Lte      func(key string, value any) *validationRules
	Eq       func(key string, value any) *validationRules
	Ne       func(key string, value any) *validationRules
	V10      func(key, tag string, param ...string) *validationRules
}{
	Required: func(key string) *validationRules {
		return newValidationRules(key).Add(TagRequired, "")
	},

	Range: func(key string, min, max int) *validationRules {
		return newValidationRules(key).
			Add(TagMin, fmt.Sprint(min)).
			Add(TagMax, fmt.Sprint(max))
	},

	Min: func(key string, min int) *validationRules {
		return newValidationRules(key).Add(TagMin, fmt.Sprint(min))
	},

	Max: func(key string, max int) *validationRules {
		return newValidationRules(key).Add(TagMax, fmt.Sprint(max))
	},

	Email: func(key string) *validationRules {
		return newValidationRules(key).Add(TagEmail, "")
	},

	URL: func(key string) *validationRules {
		return newValidationRules(key).Add(TagURL, "")
	},

	UUID: func(key string, version ...int) *validationRules {
		r := newValidationRules(key)
		if len(version) > 0 && version[0] == 4 {
			return r.Add(TagUUID4, "")
		}
		return r.Add(TagUUID, "")
	},

	Len: func(key string, length int) *validationRules {
		return newValidationRules(key).Add(TagLen, fmt.Sprint(length))
	},

	OneOf: func(key string, values ...string) *validationRules {
		return newValidationRules(key).Add(TagOneOf, strings.Join(values, " "))
	},

	Pattern: func(key, pattern string) *validationRules {
		return newValidationRules(key).Add(TagRegexp, pattern)
	},

	Gt: func(key string, value any) *validationRules {
		return newValidationRules(key).Add(TagGT, fmt.Sprint(value))
	},

	Lt: func(key string, value any) *validationRules {
		return newValidationRules(key).Add(TagLT, fmt.Sprint(value))
	},

	Gte: func(key string, value any) *validationRules {
		return newValidationRules(key).Add(TagGTE, fmt.Sprint(value))
	},

	Lte: func(key string, value any) *validationRules {
		return newValidationRules(key).Add(TagLTE, fmt.Sprint(value))
	},

	Eq: func(key string, value any) *validationRules {
		return newValidationRules(key).Add(TagEQ, fmt.Sprint(value))
	},

	Ne: func(key string, value any) *validationRules {
		return newValidationRules(key).Add(TagNE, fmt.Sprint(value))
	},

	V10: func(key, tag string, param ...string) *validationRules {
		r := newValidationRules(key)
		if len(param) > 0 {
			return r.Add(tag, param[0])
		}
		return r.Add(tag, "")
	},
}
