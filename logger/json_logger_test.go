package logger

import (
	"context"
	"fmt"
	"os"
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
