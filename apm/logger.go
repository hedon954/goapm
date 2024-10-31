package apm

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hedon954/goapm/apm/internal"
)

const traceID = "trace_id"

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.AddHook(&logrusHook{})
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
	if span := trace.SpanFromContext(ctx); span != nil {
		kv[traceID] = span.SpanContext().TraceID().String()
		span.SetAttributes(attribute.Bool("error", true))
		span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
	}

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
