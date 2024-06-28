package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
)

type container struct {
	Test  string `json:"test"`
	Inner *container
}

func TestLogger(t *testing.T) {
	logger, _ := NewJsonLogger(context.Background(), os.Stdout, "MyApp", "MainScope", "", DEBUG, []string{TraceID})
	logger.Log("This is a log message")

	fmt.Println("-------------")
	logger.With("userID", 123).Error("This is an error with userID")

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
