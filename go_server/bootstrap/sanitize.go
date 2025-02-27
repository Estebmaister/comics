package bootstrap

import (
	"reflect"
	"strings"
	"time"
)

var (
	// Add sensitive field patterns (in lowercase for simplicity)
	sensitivePatterns = []string{"secret", "password", "pass", "token", "key"}
)

// Sanitize is the public entry point that sanitizes the provided value.
// It initializes a fresh visited map so that calls are independent.
func Sanitize(v any) any {
	visited := make(map[uintptr]bool)
	return sanitize(v, &visited)
}

// sanitize recursively traverses v, redacting sensitive fields.
// The visited map tracks pointer addresses to avoid infinite loops.
func sanitize(v any, visited *map[uintptr]bool) any {
	if *visited == nil {
		*visited = make(map[uintptr]bool)
	}

	val := reflect.ValueOf(v)

	// Handle nil values
	if !val.IsValid() {
		return nil
	}

	// If the value is a pointer, check for cycles and dereference.
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		ptr := val.Pointer()
		if (*visited)[ptr] {
			return "**CYCLIC_REFERENCE**"
		}
		(*visited)[ptr] = true
		defer delete(*visited, ptr)
		return sanitize(val.Elem().Interface(), visited)
	}

	// Special case for time.Time: return as is
	if _, ok := v.(time.Time); ok {
		return v
	}

	switch val.Kind() {
	case reflect.Struct:
		return sanitizeStruct(val, *visited)
	case reflect.Slice, reflect.Array:
		length := val.Len()
		resultSlice := make([]any, length)
		for i := range length {
			resultSlice[i] = sanitize(val.Index(i).Interface(), visited)
		}
		return resultSlice
	case reflect.Map:
		resultMap := make(map[any]any)
		for _, key := range val.MapKeys() {
			// Assuming keys are not sensitive; otherwise, process keys too.
			resultMap[sanitize(key.Interface(), visited)] = sanitize(val.MapIndex(key).Interface(), visited)
		}
		return resultMap
	case reflect.String:
		// Without context (field name), we simply return the string.
		return val.String()
	default:
		return v
	}
}

// sanitizeStruct traverses the fields of a struct and sanitizes them.
// It uses the field name to determine if the value should be redacted.
func sanitizeStruct(val reflect.Value, visited map[uintptr]bool) map[string]any {
	t := val.Type()
	result := make(map[string]any)

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := t.Field(i)
		fieldName := fieldType.Name

		// Skip unexported fields.
		if !field.CanInterface() {
			continue
		}

		// Check if field name is sensitive.
		if isSensitive(fieldName) {
			// Check the field's type to decide how to redact.
			switch field.Kind() {
			case reflect.String:
				s := field.String()
				prefix := ""
				if len(s) >= 2 {
					prefix = s[:2]
				} else {
					prefix = s
				}
				result[fieldName] = prefix + "**REDACTED**"
			case reflect.Int64:
				// Check if it's a time.Duration.
				if field.Type() == reflect.TypeOf(time.Duration(0)) {
					dur := field.Interface().(time.Duration).String()
					prefix := ""
					if len(dur) >= 2 {
						prefix = dur[:2]
					} else {
						prefix = dur
					}
					result[fieldName] = prefix + "**REDACTED**"
				} else {
					result[fieldName] = "**REDACTED**"
				}
			default:
				result[fieldName] = "**REDACTED**"
			}
			continue
		}

		// Otherwise, recursively sanitize the field.
		result[fieldName] = sanitize(field.Interface(), &visited)
	}

	return result
}

// isSensitive checks if the fieldName contains any sensitive patterns.
func isSensitive(fieldName string) bool {
	lowerName := strings.ToLower(fieldName)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerName, pattern) {
			return true
		}
	}
	return false
}
