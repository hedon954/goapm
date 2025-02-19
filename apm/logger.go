package apm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hedon954/goapm/internal"
)

const traceID = "trace_id"

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
	if parentSpan := trace.SpanFromContext(entry.Context); parentSpan != nil {
		_, span := parentSpan.TracerProvider().Tracer("error-logger").Start(
			entry.Context, "log.error",
		)
		defer span.End()

		entry.Data[traceID] = span.SpanContext().TraceID().String()
		span.SetAttributes(attribute.Bool("error", true))
		span.RecordError(getEntryError(entry), trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
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
