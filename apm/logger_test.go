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

// helper functions to create different stack depths
func depth1() string  { return findCaller() }
func depth2() string  { return depth1() }
func depth3() string  { return depth2() }
func depth4() string  { return depth3() }
func depth5() string  { return depth4() }
func depth6() string  { return depth5() }
func depth7() string  { return depth6() }
func depth8() string  { return depth7() }
func depth9() string  { return depth8() }
func depth10() string { return depth9() }

func BenchmarkFindCaller(b *testing.B) {
	tests := []struct {
		name     string
		callFunc func() string
	}{
		{"Depth1", depth1},
		{"Depth2", depth2},
		{"Depth3", depth3},
		{"Depth4", depth4},
		{"Depth5", depth5},
		{"Depth6", depth6},
		{"Depth7", depth7},
		{"Depth8", depth8},
		{"Depth9", depth9},
		{"Depth10", depth10},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = tt.callFunc()
			}
		})
	}
}
