package mongoutil

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// BSONType constrains types that can be directly extracted from BSON documents.
type BSONType interface {
	string | bool | int | int32 | int64 | time.Time
}

// Get extracts a typed value from a bson.M document.
// Returns zero value if the key is missing or type assertion fails.
// Automatically handles numeric type coercion between int/int32/int64
// and time coercion from bson.DateTime to time.Time.
func Get[T BSONType](m bson.M, key string) T {
	var zero T

	v, ok := m[key]
	if !ok {
		return zero
	}

	// Direct type assertion (fast path)
	if typed, ok := v.(T); ok {
		return typed
	}

	// Time coercion for bson.DateTime -> time.Time
	if _, isTime := any(zero).(time.Time); isTime {
		return any(coerceTime(v)).(T)
	}

	// Numeric coercion for int/int32/int64
	return coerceNumeric[T](v)
}

// GetPtr extracts a typed pointer from a bson.M document.
// Returns nil if the key is missing or type assertion fails.
// Returns a pointer to the value if extraction succeeds.
// Handles time coercion from bson.DateTime to time.Time.
func GetPtr[T BSONType](m bson.M, key string) *T {
	v, ok := m[key]
	if !ok {
		return nil // Missing key
	}

	// Direct type assertion
	if typed, ok := v.(T); ok {
		return &typed
	}

	var zero T

	// Time coercion for bson.DateTime -> time.Time
	if _, isTime := any(zero).(time.Time); isTime {
		result := any(coerceTime(v)).(T)
		if any(result) == any(zero) {
			return nil // Coercion failed (zero time)
		}
		return &result
	}

	// Numeric coercion
	result := coerceNumeric[T](v)
	if any(result) == any(zero) {
		return nil // Coercion failed
	}
	return &result
}

// GetSlice extracts a typed slice from a bson.M document.
// Returns nil if the key is missing or the value is not a bson.A.
// Only includes elements that successfully type assert to T.
// Handles time coercion from bson.DateTime to time.Time.
//
// Note: Manual iteration is required because Go cannot cast []interface{} to []T directly.
// This is a language limitation, not a design choice.
func GetSlice[T BSONType](m bson.M, key string) []T {
	v, ok := m[key]
	if !ok {
		return nil
	}

	arr, ok := v.(bson.A)
	if !ok {
		return nil
	}

	var zero T
	_, isTime := any(zero).(time.Time)

	result := make([]T, 0, len(arr))
	for _, item := range arr {
		// Direct type assertion
		if typed, ok := item.(T); ok {
			result = append(result, typed)
			continue
		}

		// Time coercion for bson.DateTime -> time.Time
		if isTime {
			coerced := any(coerceTime(item)).(T)
			if any(coerced) != any(zero) {
				result = append(result, coerced)
			}
			continue
		}

		// Numeric coercion for int/int32/int64
		coerced := coerceNumeric[T](item)
		if any(coerced) != any(zero) {
			result = append(result, coerced)
		}
	}

	return result
}

// coerceNumeric handles automatic type conversion between int, int32, and int64.
// Returns zero value if the input cannot be converted to the target type T.
func coerceNumeric[T BSONType](v interface{}) T {
	var zero T

	// Determine target type using type switch on zero value
	switch any(zero).(type) {
	case int:
		return any(toInt(v)).(T)
	case int32:
		return any(toInt32(v)).(T)
	case int64:
		return any(toInt64(v)).(T)
	default:
		// Non-numeric type - return zero value
		return zero
	}
}

// toInt converts various numeric types to int.
func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	default:
		return 0
	}
}

// toInt32 converts various numeric types to int32.
func toInt32(v interface{}) int32 {
	switch val := v.(type) {
	case int32:
		return val
	case int:
		return int32(val)
	case int64:
		return int32(val)
	default:
		return 0
	}
}

// toInt64 converts various numeric types to int64.
func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	default:
		return 0
	}
}

// coerceTime converts bson.DateTime or ISO 8601 string to time.Time.
// Returns zero time if the input cannot be converted.
func coerceTime(v interface{}) time.Time {
	switch val := v.(type) {
	case bson.DateTime:
		return val.Time()
	case string:
		// ISO 8601 string parsing (handles timestamps stored as strings)
		t, err := time.Parse(time.RFC3339Nano, val)
		if err != nil {
			t, err = time.Parse(time.RFC3339, val)
			if err != nil {
				return time.Time{}
			}
		}
		return t
	default:
		return time.Time{}
	}
}
