package apm

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hedon954/goapm/internal"
)

const (
	traceID          = "trace_id"
	logrusTracerName = "goapm/logrus"

	emptyTraceID = "00000000000000000000000000000000"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.AddHook(&logrusHook{})
	logrus.AddHook(&logrusTracerHook{})
}

type logger struct{}

var Logger = &logger{}

func (l *logger) Info(ctx context.Context, action string, kv map[string]any) {
	logrus.
		WithContext(ctx).
		WithFields(kv).
		Info(action)
}

func (l *logger) Debug(ctx context.Context, action string, kv map[string]any) {
	logrus.
		WithContext(ctx).
		WithFields(kv).
		Debug(action)
}

func (l *logger) Error(ctx context.Context, action string, err error, kv map[string]any) {
	if kv == nil {
		kv = make(map[string]any)
	}
	kv["err"] = err

	logrus.
		WithContext(ctx).
		WithFields(kv).
		Error(action)
}

func (l *logger) Warn(ctx context.Context, action string, kv map[string]any) {
	logrus.
		WithContext(ctx).
		WithFields(kv).
		Warn(action)
}

type logrusHook struct{}

func (l *logrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (l *logrusHook) Fire(entry *logrus.Entry) error {
	entry.Data["host"] = internal.BuildInfo.Hostname()
	entry.Data["app"] = internal.BuildInfo.AppName()
	return nil
}

type logrusTracerHook struct{}

func (l *logrusTracerHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel}
}

func (l *logrusTracerHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}
	spanCtx := trace.SpanContextFromContext(entry.Context)
	if !spanCtx.IsValid() {
		return nil
	}

	tracer := otel.Tracer(logrusTracerName)
	_, span := tracer.Start(entry.Context, "log.error")
	defer span.End()

	traceID := span.SpanContext().TraceID().String()
	if traceID != emptyTraceID {
		entry.Data[traceID] = traceID
	}

	span.SetAttributes(attribute.Bool("error", true))
	span.RecordError(getEntryError(entry), trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
	if caller := findCaller(); caller != "" {
		span.SetAttributes(attribute.String("caller", caller))
	}
	return nil
}

func getEntryError(entry *logrus.Entry) error {
	if errField, exists := entry.Data["err"]; exists {
		if e, ok := errField.(error); ok {
			return e
		}
		return fmt.Errorf("%v", errField)
	}
	return errors.New(entry.Message)
}

// findCaller gets the business function where invoke logrus.Error()
func findCaller() string {
	// github.com/hedon954/goapm/apm.(*logrusTracerHook).Fire
	// github.com/sirupsen/logrus.LevelHooks.Fire
	// github.com/sirupsen/logrus.(*Entry).fireHooks
	// github.com/sirupsen/logrus.(*Entry).log
	// github.com/sirupsen/logrus.(*Entry).Log
	// github.com/sirupsen/logrus.(*Entry).Error
	const startDepth = 6
	const maxStackDepth = 15

	for i := startDepth; i < startDepth+maxStackDepth; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		fnName := fn.Name()
		if strings.Contains(fnName, "logrus") ||
			strings.Contains(fnName, "Entry") ||
			strings.Contains(file, "logrus") {
			continue
		}

		if idx := strings.LastIndex(fnName, "."); idx >= 0 {
			fnName = fnName[idx+1:]
		}
		if fnName == "Error" {
			continue
		}

		return fmt.Sprintf("%s:%d %s", file, line, fnName)
	}
	return ""
}
