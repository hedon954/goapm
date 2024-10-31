package apm

import (
	"fmt"
	"net/http"
	"runtime/debug"
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

func OtelMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

// GinOtel creates a Gin middleware for tracing, metrics and logging.
func GinOtel() gin.HandlerFunc {
	tracer := otel.Tracer(ginTracerName)

	return func(c *gin.Context) {
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
				span.SetAttributes(attribute.Bool("haserror", true))
				span.RecordError(
					fmt.Errorf("%v", err),
					trace.WithStackTrace(true),
					trace.WithTimestamp(time.Now()),
				)

				// log
				Logger.Error(ctx, "panic in gin http handler", fmt.Errorf("panic: %v", err), map[string]any{
					"method": c.Request.Method,
					"path":   c.FullPath(),
					"params": c.Request.Form.Encode(),
					"stack":  string(debug.Stack()),
				})
				c.AbortWithStatus(http.StatusInternalServerError)
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

			// metrics
			serverHandleHistogram.WithLabelValues(
				MetricTypeHTTP, c.Request.Method+"."+c.FullPath(), strconv.Itoa(status), "", "",
			).Observe(elapsed.Seconds())
		}()

		// handle request
		c.Next()
	}
}
