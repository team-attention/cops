package mongoutil_test

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/team-attention/cops/api/internal/platform/util/mongoutil"
)

func TestGet_String(t *testing.T) {
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		expected string
	}{
		{
			name:     "valid string",
			doc:      bson.M{"name": "alice"},
			key:      "name",
			expected: "alice",
		},
		{
			name:     "missing key",
			doc:      bson.M{},
			key:      "name",
			expected: "",
		},
		{
			name:     "type mismatch",
			doc:      bson.M{"name": 123},
			key:      "name",
			expected: "",
		},
		{
			name:     "empty string",
			doc:      bson.M{"name": ""},
			key:      "name",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mongoutil.Get[string](tt.doc, tt.key)
			if result != tt.expected {
				t.Errorf("Get[string] = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGet_Bool(t *testing.T) {
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		expected bool
	}{
		{
			name:     "true value",
			doc:      bson.M{"active": true},
			key:      "active",
			expected: true,
		},
		{
			name:     "false value",
			doc:      bson.M{"active": false},
			key:      "active",
			expected: false,
		},
		{
			name:     "missing key",
			doc:      bson.M{},
			key:      "active",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mongoutil.Get[bool](tt.doc, tt.key)
			if result != tt.expected {
				t.Errorf("Get[bool] = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGet_NumericCoercion(t *testing.T) {
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		wantInt  int
		wantI32  int32
		wantI64  int64
	}{
		{
			name:     "int value",
			doc:      bson.M{"count": int(42)},
			key:      "count",
			wantInt:  42,
			wantI32:  42,
			wantI64:  42,
		},
		{
			name:     "int32 value",
			doc:      bson.M{"count": int32(100)},
			key:      "count",
			wantInt:  100,
			wantI32:  100,
			wantI64:  100,
		},
		{
			name:     "int64 value",
			doc:      bson.M{"count": int64(200)},
			key:      "count",
			wantInt:  200,
			wantI32:  200,
			wantI64:  200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mongoutil.Get[int](tt.doc, tt.key); got != tt.wantInt {
				t.Errorf("Get[int] = %v, want %v", got, tt.wantInt)
			}
			if got := mongoutil.Get[int32](tt.doc, tt.key); got != tt.wantI32 {
				t.Errorf("Get[int32] = %v, want %v", got, tt.wantI32)
			}
			if got := mongoutil.Get[int64](tt.doc, tt.key); got != tt.wantI64 {
				t.Errorf("Get[int64] = %v, want %v", got, tt.wantI64)
			}
		})
	}
}

func TestGet_Time(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		expected time.Time
	}{
		{
			name:     "valid time",
			doc:      bson.M{"created_at": now},
			key:      "created_at",
			expected: now,
		},
		{
			name:     "missing key",
			doc:      bson.M{},
			key:      "created_at",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mongoutil.Get[time.Time](tt.doc, tt.key)
			if !result.Equal(tt.expected) {
				t.Errorf("Get[time.Time] = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetPtr_String(t *testing.T) {
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		wantNil  bool
		wantVal  string
	}{
		{
			name:    "valid string",
			doc:     bson.M{"name": "alice"},
			key:     "name",
			wantNil: false,
			wantVal: "alice",
		},
		{
			name:    "missing key returns nil",
			doc:     bson.M{},
			key:     "name",
			wantNil: true,
		},
		{
			name:    "type mismatch returns nil",
			doc:     bson.M{"name": 123},
			key:     "name",
			wantNil: true,
		},
		{
			name:    "empty string returns pointer",
			doc:     bson.M{"name": ""},
			key:     "name",
			wantNil: false,
			wantVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mongoutil.GetPtr[string](tt.doc, tt.key)
			if tt.wantNil {
				if result != nil {
					t.Errorf("GetPtr[string] = %v, want nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("GetPtr[string] = nil, want non-nil")
				} else if *result != tt.wantVal {
					t.Errorf("GetPtr[string] = %q, want %q", *result, tt.wantVal)
				}
			}
		})
	}
}

func TestGetPtr_NumericCoercion(t *testing.T) {
	doc := bson.M{"count": int32(42)}

	// int32 -> int32 (direct)
	if ptr := mongoutil.GetPtr[int32](doc, "count"); ptr == nil || *ptr != 42 {
		t.Errorf("GetPtr[int32] failed")
	}

	// int32 -> int (coercion)
	if ptr := mongoutil.GetPtr[int](doc, "count"); ptr == nil || *ptr != 42 {
		t.Errorf("GetPtr[int] coercion failed")
	}

	// int32 -> int64 (coercion)
	if ptr := mongoutil.GetPtr[int64](doc, "count"); ptr == nil || *ptr != 42 {
		t.Errorf("GetPtr[int64] coercion failed")
	}
}

func TestGetSlice_String(t *testing.T) {
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		expected []string
	}{
		{
			name:     "valid string array",
			doc:      bson.M{"tags": bson.A{"alice", "bob", "charlie"}},
			key:      "tags",
			expected: []string{"alice", "bob", "charlie"},
		},
		{
			name:     "empty array",
			doc:      bson.M{"tags": bson.A{}},
			key:      "tags",
			expected: []string{},
		},
		{
			name:     "missing key",
			doc:      bson.M{},
			key:      "tags",
			expected: nil,
		},
		{
			name:     "mixed types (partial extraction)",
			doc:      bson.M{"tags": bson.A{"alice", 123, "bob"}},
			key:      "tags",
			expected: []string{"alice", "bob"},
		},
		{
			name:     "not an array",
			doc:      bson.M{"tags": "single"},
			key:      "tags",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mongoutil.GetSlice[string](tt.doc, tt.key)
			if !stringSliceEqual(result, tt.expected) {
				t.Errorf("GetSlice[string] = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetSlice_NumericCoercion(t *testing.T) {
	tests := []struct {
		name     string
		doc      bson.M
		key      string
		wantInt  []int
		wantI32  []int32
		wantI64  []int64
	}{
		{
			name:     "int array",
			doc:      bson.M{"nums": bson.A{int(1), int(2), int(3)}},
			key:      "nums",
			wantInt:  []int{1, 2, 3},
			wantI32:  []int32{1, 2, 3},
			wantI64:  []int64{1, 2, 3},
		},
		{
			name:     "int32 array",
			doc:      bson.M{"nums": bson.A{int32(10), int32(20)}},
			key:      "nums",
			wantInt:  []int{10, 20},
			wantI32:  []int32{10, 20},
			wantI64:  []int64{10, 20},
		},
		{
			name:     "mixed numeric types",
			doc:      bson.M{"nums": bson.A{int(1), int32(2), int64(3)}},
			key:      "nums",
			wantInt:  []int{1, 2, 3},
			wantI32:  []int32{1, 2, 3},
			wantI64:  []int64{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mongoutil.GetSlice[int](tt.doc, tt.key); !intSliceEqual(got, tt.wantInt) {
				t.Errorf("GetSlice[int] = %v, want %v", got, tt.wantInt)
			}
			if got := mongoutil.GetSlice[int32](tt.doc, tt.key); !int32SliceEqual(got, tt.wantI32) {
				t.Errorf("GetSlice[int32] = %v, want %v", got, tt.wantI32)
			}
			if got := mongoutil.GetSlice[int64](tt.doc, tt.key); !int64SliceEqual(got, tt.wantI64) {
				t.Errorf("GetSlice[int64] = %v, want %v", got, tt.wantI64)
			}
		})
	}
}

// Helper functions for slice comparison

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func int32SliceEqual(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func int64SliceEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
