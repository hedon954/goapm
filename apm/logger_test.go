package apm

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogrusHook(t *testing.T) {
	logrus.WithContext(context.Background()).Error("test")
}

func TestLoggerHook(t *testing.T) {
	aFuncToCallLogrusError()
	aFuncToCallLoggerError()
}

func aFuncToCallLogrusError() {
	logrus.WithContext(context.Background()).Error("test")
}

func aFuncToCallLoggerError() {
	Logger.Error(context.Background(), "test", errors.New("errmsg"), map[string]any{"a": "b"})
}
