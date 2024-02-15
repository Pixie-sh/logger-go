package logger

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestFactory(t *testing.T) {
	factory, err := NewFactory(context.Background(), DefaultFactoryConfiguration)
	assert.Nil(t, err)

	logger, err := factory.Create(context.Background(), Configuration{
		App:      "App",
		Scope:    "Scope",
		UID:      "uid",
		LogLevel: LOG,
		Driver:   JSONLoggerDriver,
		Values: JSONLoggerConfiguration{
			Writer: os.Stdout,
		},
	})
	assert.Nil(t, err)

	logger.Log("This is a log messagee")

	fmt.Println("-------------")
	logger.With("userID", 123).Error("This is an error with userID")

	fmt.Println("-------------")
	log := logger.With("A", container{Test: "A inner", Inner: &container{Test: "B inner"}})
	log.Log("something to flush the logger")
}
