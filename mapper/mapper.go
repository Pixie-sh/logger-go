package mapper

import (
	"github.com/mitchellh/mapstructure"
	"reflect"
)

// ObjectToStruct map from interface{} map[string]interface{} to respective struct
func ObjectToStruct(from interface{}, to interface{}) error {
	return mapstructure.Decode(from, to)
}

// IsComplexType checks if the value is a complex type that should be JSON marshaled.
func IsComplexType(v any) bool {
	if v == nil {
		return false
	}
	kind := reflect.TypeOf(v).Kind()
	return kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array
}
