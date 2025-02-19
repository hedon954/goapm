package apm

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	ginTracerName = "goapm/gin"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

type ginOtel struct {
	panicHooks     []func(c *gin.Context, panic any) (stop bool)
	recordResponse func(c *gin.Context) bool
}

type GinOtelOption func(o *ginOtel)

func WithPanicHook(hook func(c *gin.Context, panic any) (stop bool)) GinOtelOption {
	return func(o *ginOtel) {
		o.panicHooks = append(o.panicHooks, hook)
	}
}

func WithRecordResponse(recordResponse func(c *gin.Context) bool) GinOtelOption {
	return func(o *ginOtel) {
		o.recordResponse = recordResponse
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
			// panic recover
			if err := recover(); err != nil {
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
				span.SetAttributes(
					attribute.Bool("pinned", true),
					attribute.String("http.response.body", blw.body.String()),
					// TODO: record request body
				)
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
