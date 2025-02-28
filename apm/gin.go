package apm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	ginTracerName = "goapm/gin"

	GinTraceIDKey = "goapm/gin/trace_id"

	ginBodyKey = "goapm/gin/body"
)

// bodyLogWriter is a wrapper around gin.ResponseWriter that logs the response body.
// It is used to record the response body when needed.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write writes the response body to the buffer before writing it to the response.
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ginOtel is the middleware for tracing, metrics and logging.
type ginOtel struct {
	// panic hooks are called when a panic occurs.
	panicHooks []func(c *gin.Context, panic any, stack []byte) (stop bool)

	// recordResponse is called to determine if the response should be recorded.
	// any of recordResponse and recordResponseWhenLogrusError return true, the response will be recorded.
	recordResponse func(c *gin.Context) bool

	// recordResponseWhenLogrusError is called to determine if the response should be recorded
	// when an error attribute is set.
	recordResponseWhenLogrusError bool

	// filterRecordResponse is called to determine if the response should not be recorded.
	// if it returns true, the response will not be recorded
	// regardless of the value of recordResponse and recordResponseWhenLogrusError.
	filterRecordResponse func(c *gin.Context) bool

	// formatResponse is called to format the response body.
	formatResponse func(c *gin.Context, body *bytes.Buffer) string
}

// GinOtelOption is a function that configures the ginOtel middleware.
type GinOtelOption func(o *ginOtel)

// WithPanicHook sets a hook to be called when a panic occurs.
func WithPanicHook(hook func(c *gin.Context, panic any, stack []byte) (stop bool)) GinOtelOption {
	return func(o *ginOtel) {
		o.panicHooks = append(o.panicHooks, hook)
	}
}

// WithRecordResponse sets a function to determine if the response should be recorded.
// If it returns true, the tracer will record with the response body.
func WithRecordResponse(recordResponse func(c *gin.Context) bool) GinOtelOption {
	return func(o *ginOtel) {
		o.recordResponse = recordResponse
	}
}

// WithResponseFormat sets a function to format the response body.
// If not set, the response body will be recorded as is.
func WithResponseFormat(fn func(c *gin.Context, body *bytes.Buffer) string) GinOtelOption {
	return func(o *ginOtel) {
		o.formatResponse = fn
	}
}

// WithRecordResponseWhenLogrusError sets a function to determine if the response should be recorded
// when `logrus.WithContext(ctx).Error()` is called.
func WithRecordResponseWhenLogrusError(record bool) GinOtelOption {
	return func(o *ginOtel) {
		o.recordResponseWhenLogrusError = record
	}
}

// WithFilterRecordResponse sets a function to determine if the response should not be recorded.
// if it returns true, the response will not be recorded
// regardless of the value of recordResponse and recordResponseWhenLogrusError.
func WithFilterRecordResponse(filter func(c *gin.Context) bool) GinOtelOption {
	return func(o *ginOtel) {
		o.filterRecordResponse = filter
	}
}

// GinOtel creates a Gin middleware for tracing, metrics and logging.
func GinOtel(opts ...GinOtelOption) gin.HandlerFunc {
	tracer := otel.Tracer(ginTracerName)

	o := &ginOtel{}
	for _, opt := range opts {
		opt(o)
	}

	return func(c *gin.Context) {
		ctx := newCtxWithGin(c.Request.Context(), c)
		cacheJsonBody(c)

		// check if record response
		mayRecordResponse := o.recordResponseWhenLogrusError
		recordResponse := false
		if o.recordResponse != nil && o.recordResponse(c) {
			recordResponse = true
		}

		mustNotRecordResponse := false
		if o.filterRecordResponse != nil && o.filterRecordResponse(c) {
			mustNotRecordResponse = true
		}

		var blw *bodyLogWriter
		if !mustNotRecordResponse && (mayRecordResponse || recordResponse) {
			blw = &bodyLogWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
			}
			c.Writer = blw
		}

		// metrics
		serverHandleCounter.WithLabelValues(MetricTypeHTTP, c.Request.Method+"."+c.FullPath(), "", "").Inc()

		// trace
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request.Header))
		ctx, span := tracer.Start(ctx, "HTTP "+c.Request.Method+" "+c.FullPath())
		defer span.End()
		c.Request = c.Request.WithContext(ctx)
		c.Set(GinTraceIDKey, span.SpanContext().TraceID().String())
		c.Writer.Header().Set(GinTraceIDKey, span.SpanContext().TraceID().String())

		start := time.Now()
		defer func() {

			// panic recover
			hasPanic := false
			if err := recover(); err != nil {
				hasPanic = true

				span.SetAttributes(
					attribute.Bool("error", true),
					attribute.String("http.request.path", c.FullPath()),
					attribute.String("http.request.method", c.Request.Method),
				)
				setRequestParams(c, span)
				span.RecordError(
					fmt.Errorf("%v", err),
					trace.WithStackTrace(true),
					trace.WithTimestamp(time.Now()),
				)
				c.AbortWithStatus(http.StatusInternalServerError)

				// run panic hooks
				stack := getStack()
				for _, hook := range o.panicHooks {
					if hook(c, err, stack) {
						break
					}
				}
			}

			// http response status code
			status := c.Writer.Status()
			elapsed := time.Since(start)
			span.SetAttributes(
				attribute.Int("http.response.code", status),
				attribute.Int64("http.duration_ms", elapsed.Milliseconds()),
			)

			// business error code
			businessErrorCode := c.Writer.Header().Get(HeaderBusinessErrorCode)
			businessErrorMsg := c.Writer.Header().Get(HeaderBusinessErrorMsg)
			if businessErrorCode != "" {
				span.SetAttributes(
					attribute.String("http.response.business_error_code", businessErrorCode),
					attribute.String("http.response.business_error_msg", businessErrorMsg),
				)
			}

			// check if any `logrus.WithContext(ctx).Error()`` is called
			if !mustNotRecordResponse && !recordResponse && mayRecordResponse {
				if _, ok := c.Get(errorLogKey); ok {
					recordResponse = true
				}
			}

			// record response
			if !mustNotRecordResponse && recordResponse {
				span.SetAttributes(attribute.Bool("pinned", true))
				if o.formatResponse != nil {
					span.SetAttributes(attribute.String("http.response.body", o.formatResponse(c, blw.body)))
				} else {
					span.SetAttributes(attribute.String("http.response.body", blw.body.String()))
				}
				if !hasPanic {
					setRequestParams(c, span)
				}
			}

			// metrics
			serverHandleHistogram.WithLabelValues(
				MetricTypeHTTP, c.Request.Method+"."+c.FullPath(), strconv.Itoa(status), "", "",
			).Observe(elapsed.Seconds())
		}()

		// handle request
		c.Next()
	}
}

//nolint:staticcheck
func newCtxWithGin(ctx context.Context, c *gin.Context) context.Context {
	return context.WithValue(ctx, gin.ContextKey, c)
}

func setRequestParams(c *gin.Context, span trace.Span) {
	span.SetAttributes(attribute.String("http.request.query", formatRequestQuery(c.Request.URL.Query())))

	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if contentType == "application/x-www-form-urlencoded" || contentType == "multipart/form-data" {
		span.SetAttributes(attribute.String("http.request.params", formatRequestParams(c.Request.Form)))
	} else if contentType == "application/json" {
		span.SetAttributes(attribute.String("http.request.body", c.GetString(ginBodyKey)))
	}
}

func formatRequestQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}
	param := make(map[string]string, len(query))
	for k, v := range query {
		if len(v) > 0 {
			param[k] = v[0]
		}
	}
	bs, _ := sonic.Marshal(param)
	return string(bs)
}

func formatRequestParams(form url.Values) string {
	param := make(map[string]string, len(form))
	for k, v := range form {
		if len(v) > 0 {
			param[k] = v[0]
		}
	}
	bs, _ := sonic.Marshal(param)
	return string(bs)
}

func getStack() []byte {
	// skip runtime.Callers, getStack, apm.GinOtel.defer func, apm.GinOtel
	const skip = 4

	pc := make([]uintptr, 50)
	n := runtime.Callers(skip, pc)
	if n == 0 {
		return []byte{}
	}

	pc = pc[:n]
	frames := runtime.CallersFrames(pc)

	var buf bytes.Buffer
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&buf, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}
	return buf.Bytes()
}

func cacheJsonBody(c *gin.Context) {
	contentType := strings.ToLower(c.Request.Header.Get("Content-Type"))
	if contentType == "application/json" {
		body := c.Request.Body
		if body != nil {
			bodyBytes, _ := io.ReadAll(body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			c.Set(ginBodyKey, string(bodyBytes))
		}
	}
}
