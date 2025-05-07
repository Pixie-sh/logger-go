package logger

import (
	"encoding/base64"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/pixie-sh/logger-go/structs"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	JSONLoggerDriver = "json_logger_driver"
	TextLoggerDriver = "text_logger_driver"
)

var DefaultTextParser = func(
	level LogLevelEnum,
	app string,
	scope string,
	expandedMsg string,
	logVersion string,
	ctxLog any,
	fields map[string]any,
) []byte {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	logLine := fmt.Sprintf("[%s][%s]$_ %s {%s,%s,%s}",
		timestamp,
		level.String(),
		expandedMsg,
		scope,
		app,
		logVersion,
	)

	if len(fields) > 0 {
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := fields[k]
			if v == nil {
				logLine += fmt.Sprintf("\n  Fields.%s: nil", k)
			} else if err, ok := v.(error); ok {
				logLine += fmt.Sprintf("\n  Fields.%s: \"%s\"", k, err.Error())
			} else {
				// Check if it's a struct or map to flatten it
				switch reflect.ValueOf(v).Kind() {
				case reflect.Struct, reflect.Map, reflect.Ptr:
					flattenAndAppendFields(k, v, &logLine, "Fields")
				default:
					logLine += fmt.Sprintf("\n  Fields.%s: %s", k, formatValueForText(v))
				}
			}
		}
	}

	if ctxLog != nil {
		// For context, we want to flatten the map
		if mapCtx, ok := ctxLog.(map[string]interface{}); ok {
			keys := make([]string, 0, len(mapCtx))
			for k := range mapCtx {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				v := mapCtx[k]
				logLine += fmt.Sprintf("\n  Context.%s: %s", k, formatValueForText(v))
			}
		} else {
			logLine += fmt.Sprintf("\n  Context: %v", formatValueForText(ctxLog))
		}
	}

	return structs.UnsafeBytes(logLine)
}

// Helper function to flatten nested structures
func flattenAndAppendFields(key string, value interface{}, logLine *string, prefix string) {
	v := reflect.ValueOf(value)

	// If it's a pointer, get the underlying value
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	// Process based on kind
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.IsExported() {
				fieldValue := v.Field(i).Interface()
				fieldKey := fmt.Sprintf("%s.%s.%s", prefix, key, field.Name)

				// Check if field value is itself a struct
				if reflect.ValueOf(fieldValue).Kind() == reflect.Struct ||
					(reflect.ValueOf(fieldValue).Kind() == reflect.Ptr && !reflect.ValueOf(fieldValue).IsNil()) {
					flattenAndAppendFields(key+"."+field.Name, fieldValue, logLine, prefix)
				} else {
					*logLine += fmt.Sprintf("\n  %s: %s", fieldKey, formatValueForText(fieldValue))
				}
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			mapValue := v.MapIndex(k).Interface()
			mapKey := fmt.Sprintf("%s.%s.%s", prefix, key, k.String())

			if reflect.ValueOf(mapValue).Kind() == reflect.Struct ||
				reflect.ValueOf(mapValue).Kind() == reflect.Map ||
				(reflect.ValueOf(mapValue).Kind() == reflect.Ptr && !reflect.ValueOf(mapValue).IsNil()) {
				flattenAndAppendFields(key+"."+k.String(), mapValue, logLine, prefix)
			} else {
				*logLine += fmt.Sprintf("\n  %s: %s", mapKey, formatValueForText(mapValue))
			}
		}
	default:
		*logLine += fmt.Sprintf("\n  %s.%s: %s", prefix, key, formatValueForText(value))
	}
}

// Add this helper function to your code
func formatValueForText(value interface{}) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case []byte:
		// Format byte slices as base64
		return "base64(" + base64.StdEncoding.EncodeToString(v) + ")"

	case error:
		return fmt.Sprintf("error: %+v", v)

	case time.Time:
		// Format timestamps consistently
		return v.Format(time.RFC3339)

	case string:
		// Return strings directly
		return v

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		// Basic types can use standard formatting
		return fmt.Sprintf("%v", v)

	default:
		// Try reflection for more complex types
		val := reflect.ValueOf(v)

		// Handle pointers
		if val.Kind() == reflect.Ptr && !val.IsNil() {
			return formatValueForText(val.Elem().Interface())
		}

		// Handle maps specially
		if val.Kind() == reflect.Map {
			if val.Len() == 0 {
				return "{}"
			}

			var builder strings.Builder
			builder.WriteString("{\n")

			iter := val.MapRange()
			for iter.Next() {
				k := iter.Key().Interface()
				v := iter.Value().Interface()
				builder.WriteString(fmt.Sprintf("      %v: %s\n", k, formatValueForText(v)))
			}

			builder.WriteString("    }")
			return builder.String()
		}

		// Handle slices and arrays (except []byte which is handled above)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			if val.Len() == 0 {
				return "[]"
			}

			var builder strings.Builder
			builder.WriteString("[\n")

			for i := 0; i < val.Len(); i++ {
				builder.WriteString(fmt.Sprintf("      %s\n", formatValueForText(val.Index(i).Interface())))
			}

			builder.WriteString("    ]")
			return builder.String()
		}

		// Handle structs by showing field names
		if val.Kind() == reflect.Struct {
			var builder strings.Builder
			builder.WriteString("{\n")

			t := val.Type()
			for i := 0; i < val.NumField(); i++ {
				if t.Field(i).IsExported() {
					builder.WriteString(fmt.Sprintf("      %s: %s\n",
						t.Field(i).Name,
						formatValueForText(val.Field(i).Interface())))
				}
			}

			builder.WriteString("    }")
			return builder.String()
		}

		// Default fallback
		return fmt.Sprintf("%+v", v)
	}
}

var DefaultJSONParser = func(
	level LogLevelEnum,
	app string,
	scope string,
	expandedMsg string,
	logVersion string,
	ctxLog any,
	fields map[string]any,
) []byte {
	var logEntry = make(map[string]any)

	if fields != nil {
		for k, v := range fields {
			if v == nil {
				logEntry[k] = "nil"
			} else {
				switch v := v.(type) {
				case error:
					errorInfo := make(map[string]interface{})
					errorInfo["error"] = v.Error()

					var innerErr interface{} = v
					u, ok := innerErr.(interface{ Unwrap() error })
					if ok  && u != nil && u.Unwrap() != nil{
						unwraped := u.Unwrap()
						typeOfNil := reflect.TypeOf(unwraped)
						if typeOfNil != nil {
							errorInfo["error.unwrap"] = unwraped.Error()
						}
					}

					logEntry[k] = errorInfo
				default:
					logEntry[k] = v
				}
			}
		}
	}

	logEntry["timestamp"] = time.Now().Format(time.RFC3339)
	logEntry["level"] = level.String()
	logEntry["app"] = app
	logEntry["scope"] = scope
	logEntry["message"] = expandedMsg
	logEntry["version"] = logVersion

	if ctxLog != nil {
		logEntry["ctx"] = ctxLog
	}

	blob, err := json.Marshal(logEntry)
	if err != nil {
		return structs.UnsafeBytes(fmt.Sprintf("error marshaling log: %v; %+v", err, logEntry))
	}

	return blob
}
