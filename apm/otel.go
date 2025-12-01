package apm

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

func CustomerRecordError(span trace.Span, err error, withStackTrace bool, stackSkip int) {
	if err == nil {
		return
	}

	eventOpts := []trace.EventOption{
		trace.WithTimestamp(time.Now()),
		trace.WithAttributes(
			semconv.ExceptionType(reflect.TypeOf(err).String()),
			semconv.ExceptionMessage(err.Error()),
		),
	}

	if withStackTrace {
		eventOpts = append(eventOpts, trace.WithAttributes(
			semconv.ExceptionStacktrace(recordStackTrace(stackSkip)),
		))
	}

	span.AddEvent(semconv.ExceptionEventName, eventOpts...)
}

func recordStackTrace(stackSkip int) string {
	const MAX_STACK_TRACE_LINES = 10

	pc := make([]uintptr, MAX_STACK_TRACE_LINES)
	n := runtime.Callers(stackSkip, pc)
	frames := runtime.CallersFrames(pc[:n])
	if frames == nil {
		return "get stack trace failed"
	}

	var sb strings.Builder
	count := 0
	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		fmt.Fprintf(&sb, "%s:%d\n", frame.File, frame.Line)
		fmt.Fprintf(&sb, "\t%s\n", frame.Function)
		count++
		if count >= MAX_STACK_TRACE_LINES {
			break
		}
	}
	return sb.String()
}
