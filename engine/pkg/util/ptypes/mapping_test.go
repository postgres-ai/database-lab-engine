package ptypes

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert_StringType(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{name: "string to string", input: "hello", expected: "hello"},
		{name: "empty string to string", input: "", expected: ""},
		{name: "int64 to string", input: int64(42), expected: "42"},
		{name: "negative int64 to string", input: int64(-100), expected: "-100"},
		{name: "zero int64 to string", input: int64(0), expected: "0"},
		{name: "float64 to string", input: float64(3.14), expected: "3.14"},
		{name: "zero float64 to string", input: float64(0), expected: "0.0"},
		{name: "negative float64 to string", input: float64(-1.5), expected: "-1.5"},
		{name: "true to string", input: true, expected: "true"},
		{name: "false to string", input: false, expected: "false"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Convert(tc.input, String)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvert_StringTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{name: "nil value", input: nil},
		{name: "int value", input: 42},
		{name: "slice value", input: []string{"a"}},
		{name: "map value", input: map[string]interface{}{"k": "v"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.input, String)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported string type")
		})
	}
}

func TestConvert_Int64Type(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{name: "string to int64", input: "123", expected: int64(123)},
		{name: "negative string to int64", input: "-50", expected: int64(-50)},
		{name: "zero string to int64", input: "0", expected: int64(0)},
		{name: "int64 to int64", input: int64(999), expected: int64(999)},
		{name: "float64 to int64", input: float64(7.9), expected: int64(7)},
		{name: "zero float64 to int64", input: float64(0), expected: int64(0)},
		{name: "negative float64 to int64", input: float64(-3.2), expected: int64(-3)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Convert(tc.input, Int64)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvert_Int64TypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{name: "non-numeric string", input: "abc"},
		{name: "float string", input: "3.14"},
		{name: "bool value", input: true},
		{name: "nil value", input: nil},
		{name: "slice value", input: []string{"a"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.input, Int64)
			require.Error(t, err)
		})
	}
}

func TestConvert_Float64Type(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{name: "string to float64", input: "3.14", expected: float64(3.14)},
		{name: "integer string to float64", input: "42", expected: float64(42)},
		{name: "negative string to float64", input: "-1.5", expected: float64(-1.5)},
		{name: "zero string to float64", input: "0", expected: float64(0)},
		{name: "int64 to float64", input: int64(10), expected: float64(10)},
		{name: "float64 to float64", input: float64(2.718), expected: float64(2.718)},
		{name: "zero float64 to float64", input: float64(0), expected: float64(0)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Convert(tc.input, Float64)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvert_Float64TypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{name: "non-numeric string", input: "abc"},
		{name: "bool value", input: true},
		{name: "nil value", input: nil},
		{name: "map value", input: map[string]interface{}{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.input, Float64)
			require.Error(t, err)
		})
	}
}

func TestConvert_BoolType(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{name: "true string to bool", input: "true", expected: true},
		{name: "false string to bool", input: "false", expected: false},
		{name: "1 string to bool", input: "1", expected: true},
		{name: "0 string to bool", input: "0", expected: false},
		{name: "bool true to bool", input: true, expected: true},
		{name: "bool false to bool", input: false, expected: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Convert(tc.input, Bool)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConvert_BoolTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{name: "invalid string", input: "notabool"},
		{name: "int64 value", input: int64(1)},
		{name: "float64 value", input: float64(1.0)},
		{name: "nil value", input: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.input, Bool)
			require.Error(t, err)
		})
	}
}

func TestConvert_EmptyStringIsUnset(t *testing.T) {
	tests := []struct {
		name     string
		expected Type
	}{
		{name: "int64", expected: Int64},
		{name: "float64", expected: Float64},
		{name: "bool", expected: Bool},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Convert("", tc.expected)
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestConvert_MapType(t *testing.T) {
	t.Run("valid map", func(t *testing.T) {
		input := map[string]interface{}{"key": "value", "num": 42}
		result, err := Convert(input, Map)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("empty map", func(t *testing.T) {
		input := map[string]interface{}{}
		result, err := Convert(input, Map)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})
}

func TestConvert_MapTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{name: "string value", input: "hello"},
		{name: "int64 value", input: int64(1)},
		{name: "nil value", input: nil},
		{name: "wrong map type", input: map[string]string{"k": "v"}},
		{name: "slice value", input: []string{"a"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.input, Map)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported map type")
		})
	}
}

func TestConvert_SliceType(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result, err := Convert(input, Slice)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("empty string slice", func(t *testing.T) {
		input := []string{}
		result, err := Convert(input, Slice)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("interface slice", func(t *testing.T) {
		input := []interface{}{"a", 1, true}
		result, err := Convert(input, Slice)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("empty interface slice", func(t *testing.T) {
		input := []interface{}{}
		result, err := Convert(input, Slice)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})
}

func TestConvert_SliceTypeErrors(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{name: "string value", input: "hello"},
		{name: "int64 value", input: int64(1)},
		{name: "nil value", input: nil},
		{name: "int slice", input: []int{1, 2, 3}},
		{name: "map value", input: map[string]interface{}{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Convert(tc.input, Slice)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported slice type")
		})
	}
}

func TestConvert_InvalidType(t *testing.T) {
	_, err := Convert("value", Invalid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type for conversion")
}

func TestConvert_UnknownType(t *testing.T) {
	_, err := Convert("value", Type(99))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type for conversion")
}

func TestMapKindToType(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected Type
	}{
		{name: "string kind", kind: reflect.String, expected: String},
		{name: "int64 kind", kind: reflect.Int64, expected: Int64},
		{name: "float64 kind", kind: reflect.Float64, expected: Float64},
		{name: "bool kind", kind: reflect.Bool, expected: Bool},
		{name: "map kind", kind: reflect.Map, expected: Map},
		{name: "slice kind", kind: reflect.Slice, expected: Slice},
		{name: "int kind returns invalid", kind: reflect.Int, expected: Invalid},
		{name: "ptr kind returns invalid", kind: reflect.Ptr, expected: Invalid},
		{name: "struct kind returns invalid", kind: reflect.Struct, expected: Invalid},
		{name: "chan kind returns invalid", kind: reflect.Chan, expected: Invalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, MapKindToType(tc.kind))
		})
	}
}

func TestNewPtr(t *testing.T) {
	t.Run("string pointer", func(t *testing.T) {
		result := NewPtr("hello")
		require.True(t, result.IsValid())
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, "hello", result.Elem().Interface())
	})

	t.Run("int64 pointer", func(t *testing.T) {
		result := NewPtr(int64(42))
		require.True(t, result.IsValid())
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, int64(42), result.Elem().Interface())
	})

	t.Run("float64 pointer", func(t *testing.T) {
		result := NewPtr(float64(3.14))
		require.True(t, result.IsValid())
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, float64(3.14), result.Elem().Interface())
	})

	t.Run("bool pointer", func(t *testing.T) {
		result := NewPtr(true)
		require.True(t, result.IsValid())
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, true, result.Elem().Interface())
	})

	t.Run("map pointer", func(t *testing.T) {
		input := map[string]interface{}{"key": "value"}
		result := NewPtr(input)
		require.True(t, result.IsValid())
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, input, result.Elem().Interface())
	})

	t.Run("interface slice pointer", func(t *testing.T) {
		input := []interface{}{"a", 1}
		result := NewPtr(input)
		require.True(t, result.IsValid())
		assert.Equal(t, reflect.Ptr, result.Kind())
		assert.Equal(t, input, result.Elem().Interface())
	})

	t.Run("unsupported type returns zero value", func(t *testing.T) {
		result := NewPtr(42)
		assert.False(t, result.IsValid())
	})

	t.Run("nil returns zero value", func(t *testing.T) {
		result := NewPtr(nil)
		assert.False(t, result.IsValid())
	})

	t.Run("string slice returns zero value", func(t *testing.T) {
		result := NewPtr([]string{"a", "b"})
		assert.False(t, result.IsValid())
	})
}
