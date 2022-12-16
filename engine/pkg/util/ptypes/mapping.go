// Package ptypes helps with type conversion in projections.
package ptypes

import (
	"reflect"
	"strconv"

	"github.com/pkg/errors"
)

// Type represents the type of value.
type Type int

const (
	// Invalid is the type of unsupported values.
	Invalid Type = iota
	// String is a string type.
	String
	// Int64 is an int64 type.
	Int64
	// Float64 is a float64 type.
	Float64
	// Bool is a bool type.
	Bool
	// Map is a map type.
	Map
	// Slice is a slice type.
	Slice
)

// Convert converts the value to the given type.
func Convert(value interface{}, expected Type) (interface{}, error) {
	switch expected {
	case String:
		return convertString(value)
	case Int64:
		return convertInt64(value)
	case Float64:
		return convertFloat64(value)
	case Bool:
		return convertBool(value)
	case Map:
		return convertMap(value)
	case Slice:
		return convertSlice(value)
	}

	return nil, errors.Errorf("unsupported type for conversion: %T", value)
}

func convertString(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		if v == 0 {
			return "0.0", nil
		}

		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	}

	return nil, errors.Errorf("unsupported string type: %T", value)
}

func convertInt64(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}

		return i, nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	}

	return nil, errors.Errorf("unsupported int64 type: %T", value)
}

func convertFloat64(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, err
		}

		return f, nil
	case int64:
		return float64(v), nil
	case float64:
		return v, nil
	}

	return nil, errors.Errorf("unsupported float64 type: %T", value)
}

func convertBool(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, err
		}

		return b, nil
	case bool:
		return v, nil
	}

	return nil, errors.Errorf("unsupported bool type: %T", value)
}

func convertMap(value interface{}) (interface{}, error) {
	if v, ok := value.(map[string]interface{}); ok {
		return v, nil
	}

	return nil, errors.Errorf("unsupported map type: %T", value)
}

func convertSlice(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case []string:
		return v, nil

	case []interface{}:
		return v, nil
	}

	return nil, errors.Errorf("unsupported slice type: %T", value)
}

// MapKindToType returns the type of the given kind.
func MapKindToType(kind reflect.Kind) Type {
	switch kind {
	case reflect.String:
		return String
	case reflect.Int64:
		return Int64
	case reflect.Float64:
		return Float64
	case reflect.Bool:
		return Bool
	case reflect.Map:
		return Map
	case reflect.Slice:
		return Slice
	}

	return Invalid
}

// NewPtr returns a pointer to a value of the given type.
func NewPtr(value interface{}) reflect.Value {
	switch v := value.(type) {
	case string:
		return reflect.ValueOf(&v)
	case int64:
		return reflect.ValueOf(&v)
	case float64:
		return reflect.ValueOf(&v)
	case bool:
		return reflect.ValueOf(&v)
	case map[string]interface{}:
		return reflect.ValueOf(&v)
	case []interface{}:
		return reflect.ValueOf(&v)
	}

	return reflect.Value{}
}
