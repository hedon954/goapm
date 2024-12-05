package apm

import (
	"context"
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

type ginOtel struct {
	panicHooks []func(ctx context.Context, panic any) (stop bool)
}

type GinOtelOption func(o *ginOtel)

func WithPanicHook(hook func(ctx context.Context, panic any) (stop bool)) GinOtelOption {
	return func(o *ginOtel) {
		o.panicHooks = append(o.panicHooks, hook)
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
					if hook(ctx, err) {
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

			// metrics
			serverHandleHistogram.WithLabelValues(
				MetricTypeHTTP, c.Request.Method+"."+c.FullPath(), strconv.Itoa(status), "", "",
			).Observe(elapsed.Seconds())
		}()

		// handle request
		c.Next()
	}
}
