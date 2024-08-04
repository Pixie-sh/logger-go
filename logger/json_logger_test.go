package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"sync"
	"testing"
)

type container struct {
	Test  string `json:"test"`
	Inner *container
}

type errorStruct struct {
	Msg      string `json:"msg"`
	Field    string `json:"field"`
	pvtField string `json:"pvtfield"`
}

func (receiver errorStruct) Error() string {
	return "THIS IS WRONG"
}

func TestLogger(t *testing.T) {
	logger, _ := NewJsonLogger(context.Background(), os.Stdout, "MyApp", "MainScope", "", DEBUG, []string{TraceID})
	logger.Log("This is a log message")

	fmt.Println("-------------")
	logger.With("userID", 123).With("error", fmt.Errorf("UserID is wrong")).Error("This is an error with userID")
	logger.With("userID", 123).With("error", errorStruct{
		Msg:      "UserID is wrong",
		Field:    "User.ID",
		pvtField: "un seen field",
	}).Error("This is an error with userID using an error struct")

	fmt.Println("-------------")
	log := logger.With("A", container{Test: "A inner", Inner: &container{Test: "B inner"}})
	log.Log("something to flush the logger")
}

func TestSharedInnerJsonLogConcurrency(t *testing.T) {
	var buf bytes.Buffer
	baseLogger, err := NewJsonLogger(context.Background(), &buf, "TestApp", "TestScope", "TestUID", DEBUG, []string{"requestID"})
	if err != nil {
		t.Fatalf("Failed to create JsonLogger: %v", err)
	}

	// Create a shared innerJsonLog instance
	sharedLogger := baseLogger.With("sharedField", "sharedValue")

	const goroutines = 1000
	const operationsPerRoutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerRoutine; j++ {
				// Alternate between adding fields and logging
				if j%2 == 0 {
					sharedLogger.With(fmt.Sprintf("field%d", id), j)
				} else {
					sharedLogger.Log("Test log message from goroutine %d", id)
				}
			}
		}(i)
	}

	wg.Wait()

	logs := strings.Split(buf.String(), "\n")
	logs = logs[:len(logs)-1] // Remove last empty line

	// We expect (operationsPerRoutine / 2) logs per goroutine
	expectedLogs := goroutines * (operationsPerRoutine / 2)
	if len(logs) != expectedLogs {
		t.Errorf("Expected %d log entries, got %d", expectedLogs, len(logs))
	}

	for _, log := range logs {
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(log), &logEntry); err != nil {
			t.Errorf("Failed to unmarshal log entry: %v", err)
			continue
		}

		// Check for the shared field
		if sharedValue, ok := logEntry["sharedField"].(string); !ok || sharedValue != "sharedValue" {
			t.Errorf("Expected sharedField to be 'sharedValue', got %v", logEntry["sharedField"])
		}

		// Check for goroutine-specific fields
		foundGoroutineField := false
		for k, v := range logEntry {
			if strings.HasPrefix(k, "field") {
				foundGoroutineField = true
				if _, ok := v.(float64); !ok { // JSON numbers are unmarshaled as float64
					t.Errorf("Expected goroutine field value to be a number, got %v", v)
				}
			}
		}

		if !foundGoroutineField {
			t.Errorf("Expected to find at least one goroutine-specific field, but found none")
		}

		// Check other expected fields
		expectedFields := []string{"app", "scope", "uid", "level", "timestamp", "message", "caller"}
		for _, field := range expectedFields {
			if _, ok := logEntry[field]; !ok {
				t.Errorf("Expected field %s is missing from log entry", field)
			}
		}
	}
}

func TestInnerJsonLogClone(t *testing.T) {
	// Create a buffer to capture log output
	buf := new(bytes.Buffer)

	// Create a base JsonLogger
	baseLogger, _ := NewJsonLogger(context.Background(), buf, "TestApp", "TestScope", "TestUID", DEBUG, []string{"requestID"})

	// Create an innerJsonLog
	inner := &innerJsonLog{
		JsonLogger:        baseLogger,
		Ctx:               context.WithValue(context.Background(), "requestID", "12345"),
		fields:            map[string]any{"field1": "value1"},
		expectedCtxFields: []string{"requestID"},
	}

	// Create a segment
	segment := inner.Clone()

	// Test 1: Ensure segment is a new instance
	assert.NotSame(t, inner, segment, "Clone should return a new instance")

	// Test 2: Ensure segment has the same initial values
	segmentInner, ok := segment.(*innerJsonLog)
	assert.True(t, ok, "Clone should return an *innerJsonLog")
	assert.Equal(t, inner.JsonLogger, segmentInner.JsonLogger, "JsonLogger should be the same")
	assert.Equal(t, inner.Ctx, segmentInner.Ctx, "Context should be the same")
	assert.Equal(t, inner.expectedCtxFields, segmentInner.expectedCtxFields, "Expected context fields should be the same")
	assert.Equal(t, inner.fields, segmentInner.fields, "Fields should be initially the same")

	// Test 3: Ensure modifications to segment don't affect original
	segment.With("field2", "value2")
	assert.NotContains(t, inner.fields, "field2", "Original should not contain new field")
	assert.Contains(t, segmentInner.fields, "field2", "Clone should contain new field")

	// Test 4: Log with both and check output
	inner.Log("Inner log")
	segment.Log("Clone log")

	logLines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, logLines, 2, "Should have two log lines")

	var log1, log2 map[string]interface{}
	_ = json.Unmarshal(logLines[0], &log1)
	_ = json.Unmarshal(logLines[1], &log2)

	assert.Equal(t, "Inner log", log1["message"])
	assert.Equal(t, "Clone log", log2["message"])
	assert.Contains(t, log2, "field2")
	assert.NotContains(t, log1, "field2")
}

func TestJsonLoggerClone(t *testing.T) {
	// Create a buffer to capture log output
	buf := new(bytes.Buffer)

	// Create a base JsonLogger
	baseLogger, _ := NewJsonLogger(context.Background(), buf, "TestApp", "TestScope", "TestUID", DEBUG, []string{"requestID"})

	// Create a segment
	segment := baseLogger.Clone()

	// Test 1: Ensure segment is a new instance
	assert.NotSame(t, baseLogger, segment, "Clone should return a new instance")

	// Test 2: Ensure segment has the same properties
	segmentLogger, ok := segment.(*JsonLogger)
	assert.True(t, ok, "Clone should return a *JsonLogger")
	assert.Equal(t, baseLogger.App, segmentLogger.App, "App should be the same")
	assert.Equal(t, baseLogger.Scope, segmentLogger.Scope, "Scope should be the same")
	assert.Equal(t, baseLogger.UID, segmentLogger.UID, "UID should be the same")
	assert.Equal(t, baseLogger.LogLevel, segmentLogger.LogLevel, "LogLevel should be the same")
	assert.Equal(t, baseLogger.writer, segmentLogger.writer, "Writer should be the same")
	assert.Equal(t, baseLogger.expectedCtxFields, segmentLogger.expectedCtxFields, "Expected context fields should be the same")

	// Test 3: Ensure modifications to segment don't affect original
	modifiedClone := segment.With("field", "value")
	baseLogger.Log("Base log")
	modifiedClone.Log("Modified segment log")

	logLines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, logLines, 2, "Should have two log lines")

	var log1, log2 map[string]interface{}
	_ = json.Unmarshal(logLines[0], &log1)
	_ = json.Unmarshal(logLines[1], &log2)

	assert.Equal(t, "Base log", log1["message"])
	assert.Equal(t, "Modified segment log", log2["message"])
	assert.Contains(t, log2, "field")
	assert.NotContains(t, log1, "field")
}
func TestSegmentWithAndWithCtx(t *testing.T) {
	// Create a buffer to capture log output
	buf := new(bytes.Buffer)

	// Create a base JsonLogger
	baseLogger, _ := NewJsonLogger(context.Background(), buf, "TestApp", "TestScope", "TestUID", DEBUG, []string{"requestID", "userID"})

	// Create an initial context
	initialCtx := context.WithValue(context.Background(), "requestID", "initial-request-id")

	// Add some initial fields and context to the base logger
	baseWithFields := baseLogger.With("initialField", "initialValue").WithCtx(initialCtx)

	// Create a segment from the base logger with fields
	segment := baseWithFields.Clone()

	// Modify the segment with new field and context
	newCtx := context.WithValue(context.Background(), "userID", "new-user-id")
	modifiedSegment := segment.With("newField", "newValue").WithCtx(newCtx)

	// Log with both the original and modified segment
	baseWithFields.Log("Original log")
	modifiedSegment.Log("Modified segment log")

	// Parse the log output
	logLines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, logLines, 2, "Should have two log lines")

	var originalLog, modifiedLog map[string]interface{}
	json.Unmarshal(logLines[0], &originalLog)
	json.Unmarshal(logLines[1], &modifiedLog)

	// Test assertions
	assert.Equal(t, "Original log", originalLog["message"], "Original log message should be correct")
	assert.Equal(t, "Modified segment log", modifiedLog["message"], "Modified segment log message should be correct")

	// Check fields
	assert.Equal(t, "initialValue", originalLog["initialField"], "Original log should contain initial field")
	assert.NotContains(t, originalLog, "newField", "Original log should not contain new field")

	assert.Equal(t, "initialValue", modifiedLog["initialField"], "Modified log should contain initial field")
	assert.Equal(t, "newValue", modifiedLog["newField"], "Modified log should contain new field")

	// Check context
	originalCtx, originalCtxOk := originalLog["ctx"].(map[string]interface{})
	modifiedCtx, modifiedCtxOk := modifiedLog["ctx"].(map[string]interface{})

	assert.True(t, originalCtxOk, "Original log should have a context")
	assert.True(t, modifiedCtxOk, "Modified log should have a context")

	assert.Equal(t, "initial-request-id", originalCtx["requestID"], "Original log should have initial requestID")
	assert.NotContains(t, originalCtx, "userID", "Original log should not contain userID")

	assert.Equal(t, nil, modifiedCtx["requestID"], "Modified log should have initial requestID")
	assert.Equal(t, "new-user-id", modifiedCtx["userID"], "Modified log should have new userID")
}
