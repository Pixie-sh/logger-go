package logger

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactory(t *testing.T) {
	factory, err := NewFactory(DefaultFactoryConfiguration)
	assert.Nil(t, err)

	logger, err := factory.Create(Configuration{
		App:               "App",
		Scope:             "Scope",
		UID:               "uid",
		LogLevel:          LOG,
		Driver:            JSONLoggerDriver,
		Writer:            os.Stdout,
		ExpectedCtxFields: []string{"someValKey"},
	})
	assert.Nil(t, err)

	ctx := context.WithValue(context.Background(), "someValKey", "someVal")
	ctx = context.WithValue(ctx, TraceID, "someTraceID")
	logger.WithCtx(ctx).Log("This is a log messagee")

	fmt.Println("-------------")
	logger.With("userID", 123).Error("This is an error with userID")

	fmt.Println("-------------")
	log := logger.With("A", container{Test: "A inner", Inner: &container{Test: "B inner"}})
	log.Log("something to flush the logger")
}
