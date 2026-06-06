package logger

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

const (
	JSONLoggerDriver = "json_logger_driver"
	TextLoggerDriver = "text_logger_driver"
)

// jsonEntryPool reuses the working map across log lines to avoid an
// allocation per call on the hot path.
var jsonEntryPool = sync.Pool{
	New: func() any { return make(map[string]any, 16) },
}

// DefaultJSONParser produces one JSON object per log line.
func DefaultJSONParser(e *Entry) []byte {
	entry := jsonEntryPool.Get().(map[string]any)
	defer func() {
		for k := range entry {
			delete(entry, k)
		}
		jsonEntryPool.Put(entry)
	}()

	for k, v := range e.Fields {
		if isNilish(v) {
			entry[k] = "nil"
			continue
		}
		switch v := v.(type) {
		case error:
			ei := map[string]any{"error": v.Error()}
			if u, ok := any(v).(interface{ Unwrap() error }); ok {
				if uw := u.Unwrap(); !isNilish(uw) {
					ei["error.unwrap"] = uw.Error()
				}
			}
			entry[k] = ei
		default:
			entry[k] = v
		}
	}

	entry["timestamp"] = e.Timestamp.Format(time.RFC3339)
	entry["level"] = e.Level.String()
	entry["app"] = e.App
	entry["scope"] = e.Scope
	entry["message"] = e.Message
	entry["version"] = e.UID
	if e.Caller != nil && e.Caller.Path != "" {
		entry["caller"] = e.Caller
	}
	if e.Ctx != nil {
		entry["ctx"] = e.Ctx
	}

	blob, err := json.Marshal(entry)
	if err != nil {
		return []byte(fmt.Sprintf(`{"level":"ERROR","message":"json marshal failed: %s"}`, err.Error()))
	}
	return blob
}

// DefaultTextParser emits a single human-readable line, expanding structs,
// maps, errors, etc. via reflection.
func DefaultTextParser(e *Entry) []byte {
	timestamp := e.Timestamp.Format("2006-01-02 15:04:05")

	logLine := fmt.Sprintf("[%s][%s]$_ %s {%s,%s,%s}",
		timestamp,
		e.Level.String(),
		e.Message,
		e.Scope,
		e.App,
		e.UID,
	)

	if e.Caller != nil && e.Caller.Path != "" {
		logLine += fmt.Sprintf("\n  Caller: %s", e.Caller.Path)
	}

	if len(e.Fields) > 0 {
		keys := make([]string, 0, len(e.Fields))
		for k := range e.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := e.Fields[k]
			if isNilish(v) {
				logLine += fmt.Sprintf("\n  Fields.%s: nil", k)
				continue
			}
			if err, ok := v.(error); ok {
				logLine += fmt.Sprintf("\n  Fields.%s: \"%s\"", k, err.Error())
				continue
			}
			switch reflect.ValueOf(v).Kind() {
			case reflect.Struct, reflect.Map, reflect.Pointer:
				flattenAndAppendFields(k, v, &logLine, "Fields", 0)
			default:
				logLine += fmt.Sprintf("\n  Fields.%s: %s", k, formatValueForText(v, 0))
			}
		}
	}

	if e.Ctx != nil {
		if mapCtx, ok := e.Ctx.(map[string]any); ok {
			keys := make([]string, 0, len(mapCtx))
			for k := range mapCtx {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				logLine += fmt.Sprintf("\n  Context.%s: %s", k, formatValueForText(mapCtx[k], 0))
			}
		} else {
			logLine += fmt.Sprintf("\n  Context: %v", formatValueForText(e.Ctx, 0))
		}
	}

	return []byte(logLine)
}

// maxTextDepth bounds the recursive walk in the text parser. Without it, a
// self-referential struct (parent-pointer trees, cyclic graphs) would blow
// the stack — common enough that we have to guard.
const maxTextDepth = 8

// Helper function to flatten nested structures
func flattenAndAppendFields(key string, value any, logLine *string, prefix string, depth int) {
	if depth >= maxTextDepth {
		*logLine += fmt.Sprintf("\n  %s.%s: <max depth reached>", prefix, key)
		return
	}

	v := reflect.ValueOf(value)

	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			fieldValue := v.Field(i).Interface()
			fieldKey := fmt.Sprintf("%s.%s.%s", prefix, key, field.Name)

			fv := reflect.ValueOf(fieldValue)
			if fv.Kind() == reflect.Struct ||
				(fv.Kind() == reflect.Ptr && !fv.IsNil()) {
				flattenAndAppendFields(key+"."+field.Name, fieldValue, logLine, prefix, depth+1)
			} else {
				*logLine += fmt.Sprintf("\n  %s: %s", fieldKey, formatValueForText(fieldValue, depth+1))
			}
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			mapValue := v.MapIndex(k).Interface()
			mapKey := fmt.Sprintf("%s.%s.%s", prefix, key, k.String())

			mv := reflect.ValueOf(mapValue)
			if mv.Kind() == reflect.Struct ||
				mv.Kind() == reflect.Map ||
				(mv.Kind() == reflect.Ptr && !mv.IsNil()) {
				flattenAndAppendFields(key+"."+k.String(), mapValue, logLine, prefix, depth+1)
			} else {
				*logLine += fmt.Sprintf("\n  %s: %s", mapKey, formatValueForText(mapValue, depth+1))
			}
		}
	default:
		*logLine += fmt.Sprintf("\n  %s.%s: %s", prefix, key, formatValueForText(value, depth+1))
	}
}

func formatValueForText(value any, depth int) string {
	if isNilish(value) {
		return "nil"
	}
	if depth >= maxTextDepth {
		return "<max depth reached>"
	}

	switch v := value.(type) {
	case []byte:
		return "base64(" + base64.StdEncoding.EncodeToString(v) + ")"
	case error:
		return fmt.Sprintf("error: %+v", v)
	case time.Time:
		return v.Format(time.RFC3339)
	case string:
		return v
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", v)
	}

	val := reflect.ValueOf(value)

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		return formatValueForText(val.Elem().Interface(), depth+1)
	}

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
			fmt.Fprintf(&builder, "      %v: %s\n", k, formatValueForText(v, depth+1))
		}
		builder.WriteString("    }")
		return builder.String()
	}

	if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
		if val.Len() == 0 {
			return "[]"
		}
		var builder strings.Builder
		builder.WriteString("[\n")
		for i := 0; i < val.Len(); i++ {
			fmt.Fprintf(&builder, "      %s\n", formatValueForText(val.Index(i).Interface(), depth+1))
		}
		builder.WriteString("    ]")
		return builder.String()
	}

	if val.Kind() == reflect.Struct {
		var builder strings.Builder
		builder.WriteString("{\n")
		t := val.Type()
		for i := 0; i < val.NumField(); i++ {
			if t.Field(i).IsExported() {
				fmt.Fprintf(&builder, "      %s: %s\n",
					t.Field(i).Name,
					formatValueForText(val.Field(i).Interface(), depth+1))
			}
		}
		builder.WriteString("    }")
		return builder.String()
	}

	return fmt.Sprintf("%+v", value)
}
