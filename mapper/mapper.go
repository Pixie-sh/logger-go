package mapper

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// ObjectToStruct map from interface{} map[string]interface{} to respective struct
func ObjectToStruct(from interface{}, to interface{}) error {
	if !IsPointer(to) {
		return fmt.Errorf("to %s must be pointer", reflect.TypeOf(to))
	}

	if Nil(from) {
		return fmt.Errorf("from struct must not be nil")
	}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			TagName: "json",
			Result:  to,
			Squash:  true,
		})
	if err != nil {
		return err
	}

	err = decoder.Decode(from)
	if err != nil {
		return err
	}

	return nil
}

// IsComplexType checks if the value is a complex type that should be JSON marshaled.
func IsComplexType(v any) bool {
	if v == nil {
		return false
	}
	kind := reflect.TypeOf(v).Kind()
	return kind == reflect.Struct || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array
}

// IsPointer validates if input is pointer
func IsPointer(i interface{}) bool {
	if i == nil {
		return false
	}

	return reflect.TypeOf(i).Kind() == reflect.Ptr
}

func Nil(i interface{}) bool {
	if i == nil {
		return true
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice, reflect.Func:
		return reflect.ValueOf(i).IsNil()
	}

	return false
}