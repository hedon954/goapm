package apm

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
func depth1() (string, string)  { return findCaller() }
func depth2() (string, string)  { return depth1() }
func depth3() (string, string)  { return depth2() }
func depth4() (string, string)  { return depth3() }
func depth5() (string, string)  { return depth4() }
func depth6() (string, string)  { return depth5() }
func depth7() (string, string)  { return depth6() }
func depth8() (string, string)  { return depth7() }
func depth9() (string, string)  { return depth8() }
func depth10() (string, string) { return depth9() }

type testCaller struct{}

func (t *testCaller) aFunctionCallLogrus() {
	logrus.WithContext(context.Background()).Error("test")
}

func (t *testCaller) aFunctionCallLogrus2() {
	t.aFunctionCallLogrus()
}

func TestFindCaller(t *testing.T) {
	tc := &testCaller{}
	tc.aFunctionCallLogrus()
	tc.aFunctionCallLogrus2()
}

func BenchmarkFindCaller(b *testing.B) {
	tests := []struct {
		name     string
		callFunc func() (string, string)
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
				_, _ = tt.callFunc()
			}
		})
	}
}

func TestLogrusTracerHook_can_get_gin_context_and_set_error_flag(t *testing.T) {
	c := &gin.Context{}
	c.Request = &http.Request{}

	// create a new context with gin context
	ctx := newCtxWithGin(c.Request.Context(), c)

	// no error log key in the context
	assert.Nil(t, c.Value(errorLogKey))

	// set error log key in the context and check if it's set
	logrus.WithContext(ctx).Error("test")
	assert.True(t, c.Value(errorLogKey).(bool))

	// once error log key is set, it will be set to the context in whole trace
	logrus.WithContext(ctx).Info("test")
	assert.True(t, c.Value(errorLogKey).(bool))
	logrus.WithContext(ctx).Warn("test")
	assert.True(t, c.Value(errorLogKey).(bool))
	logrus.WithContext(ctx).Debug("test")
	assert.True(t, c.Value(errorLogKey).(bool))
	logrus.WithContext(ctx).Error("test")
	assert.True(t, c.Value(errorLogKey).(bool))
}
