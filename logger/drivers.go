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

	logLine := fmt.Sprintf("[%s] msg: %s | %s | %s | %s",
		level.String(),
		expandedMsg,
		scope,
		app,
		timestamp,
	)

	if logVersion != "" {
		logLine += fmt.Sprintf(" | %s", logVersion)
	}

	if len(fields) > 0 {
		logLine += "\n  Fields:"
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := fields[k]
			if v == nil {
				logLine += fmt.Sprintf("\n    %s: nil", k)
			} else if err, ok := v.(error); ok {
				logLine += fmt.Sprintf("\n    %s: ERROR - %s", k, err.Error())
			} else {
				logLine += fmt.Sprintf("\n    %s: %s", k, formatValueForText(v))
			}
		}
	}

	if ctxLog != nil {
		logLine += "\n  Context:"
		ctxValue := reflect.ValueOf(ctxLog)
		if ctxValue.Kind() == reflect.Ptr {
			ctxValue = ctxValue.Elem()
		}

		if ctxValue.Kind() == reflect.Struct {
			for i := 0; i < ctxValue.NumField(); i++ {
				field := ctxValue.Type().Field(i)
				if field.IsExported() {
					logLine += fmt.Sprintf("\n    %s: %v", field.Name, ctxValue.Field(i).Interface())
				}
			}
		} else {
			logLine += fmt.Sprintf("\n    %v", ctxLog)
		}
	}

	return structs.UnsafeBytes(logLine + "\n")
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
					// Create a map to hold both struct values and error string
					errorInfo := make(map[string]interface{})

					// Always add the error string
					errorInfo["error.string"] = v.Error()

					// Try to unwrap the error
					var innerErr interface{} = v
					for {
						u, ok := innerErr.(interface{ Unwrap() error })
						if !ok {
							break
						}
						innerErr = u.Unwrap()
						if innerErr == nil {
							break
						}
					}

					// check if it's a fmt.Errorf type
					if reflect.TypeOf(innerErr).String() != "*errors.errorString" {
						// for other error types, try reflection
						errorValue := reflect.ValueOf(innerErr)
						if errorValue.Kind() == reflect.Ptr {
							errorValue = errorValue.Elem()
						}
						if errorValue.Kind() == reflect.Struct {
							for i := 0; i < errorValue.NumField(); i++ {
								field := errorValue.Type().Field(i)
								if field.IsExported() {
									errorInfo[field.Name] = errorValue.Field(i).Interface()
								}
							}
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

	if logVersion != "" {
		logEntry["version"] = logVersion
	}

	if ctxLog != nil {
		logEntry["ctx"] = ctxLog
	}

	blob, err := json.Marshal(logEntry)
	if err != nil {
		return structs.UnsafeBytes(fmt.Sprintf("error marshaling log: %v; %+v", err, logEntry))
	}

	return blob
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
		// Special handling for errors
		return "ERROR: " + v.Error()

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