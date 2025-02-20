package apm

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	panicHooks     []func(c *gin.Context, panic any) (stop bool)
	recordResponse func(c *gin.Context) bool
	formatResponse func(body *bytes.Buffer) string
}

// GinOtelOption is a function that configures the ginOtel middleware.
type GinOtelOption func(o *ginOtel)

// WithPanicHook sets a hook to be called when a panic occurs.
func WithPanicHook(hook func(c *gin.Context, panic any) (stop bool)) GinOtelOption {
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
func WithResponseFormat(fn func(body *bytes.Buffer) string) GinOtelOption {
	return func(o *ginOtel) {
		o.formatResponse = fn
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
		// check if record response
		recordResponse := false
		var blw *bodyLogWriter
		if o.recordResponse != nil && o.recordResponse(c) {
			blw = &bodyLogWriter{
				ResponseWriter: c.Writer,
				body:           &bytes.Buffer{},
			}
			c.Writer = blw
			recordResponse = true
		}

		// metrics
		serverHandleCounter.WithLabelValues(MetricTypeHTTP, c.Request.Method+"."+c.FullPath(), "", "").Inc()

		// trace
		ctx := c.Request.Context()
		ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(c.Request.Header))
		ctx, span := tracer.Start(ctx, "HTTP "+c.Request.Method+" "+c.FullPath())
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		defer func() {
			hasPanic := false

			// panic recover
			if err := recover(); err != nil {
				hasPanic = true

				span.SetAttributes(
					attribute.Bool("error", true),
					attribute.String("path", c.FullPath()),
					attribute.String("method", c.Request.Method),
					attribute.String("params", c.Request.Form.Encode()),
				)
				span.RecordError(
					fmt.Errorf("%v", err),
					trace.WithStackTrace(true),
					trace.WithTimestamp(time.Now()),
				)
				c.AbortWithStatus(http.StatusInternalServerError)

				// run panic hooks
				for _, hook := range o.panicHooks {
					if hook(c, err) {
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

			// record response
			if recordResponse {
				span.SetAttributes(attribute.Bool("pinned", true))
				if o.formatResponse != nil {
					span.SetAttributes(attribute.String("response", o.formatResponse(blw.body)))
				} else {
					span.SetAttributes(attribute.String("response", blw.body.String()))
				}
				if !hasPanic {
					span.SetAttributes(attribute.String("params", formatRequestParams(c.Request.Form)))
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
