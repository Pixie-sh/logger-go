package logger

import (
	"reflect"
	"time"
)

const (
	JSONLoggerDriver = "json_logger_driver"
)

var DefaultJSONParser ParserFn = func(
	level LogLevelEnum,
	app string,
	scope string,
	expandedMsg string,
	logUID string,
	ctxLog any,
	fields map[string]any,
) map[string]any {
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

	if logUID != "" {
		logEntry["uid"] = logUID
	}

	if ctxLog != nil {
		logEntry["ctx"] = ctxLog
	}

	return logEntry
}
