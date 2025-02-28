package apm

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTracingTest setups tracing test
func setupTracingTest() (*tracetest.InMemoryExporter, *sdktrace.TracerProvider, *gin.Engine) {
	// create a memory exporter to store spans
	exporter := tracetest.NewInMemoryExporter()

	// create a trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
	)

	// set the trace provider as the global tracer provider
	otel.SetTracerProvider(tp)

	router := gin.Default()
	return exporter, tp, router
}

// hasResponseBodyAttribute checks if the spans have the attribute http.response.body
func hasResponseBodyAttribute(spans []sdktrace.ReadOnlySpan) bool {
	if len(spans) == 0 {
		return false
	}

	for _, span := range spans {
		for _, attr := range span.Attributes() {
			if attr.Key == "http.response.body" {
				return true
			}
		}
	}
	return false
}

func init() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
}

func TestGinServer_Handle(t *testing.T) {
	router := gin.Default()
	router.Use(GinOtel())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	listener, err := net.Listen("tcp", ":") //nolint:gosec
	if err != nil {
		panic(err)
	}

	go func() {
		if err := router.RunListener(listener); err != nil {
			panic(err)
		}
	}()

	for {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get("http://" + listener.Addr().String() + "/")
		if err != nil {
			continue
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatalf("Failed to close response body: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			break
		}
	}
}

func TestGetStack(t *testing.T) {
	callGetStack()
}

func callGetStack() {
	a := func() {
		stack := getStack()
		fmt.Println(string(stack))
	}
	a()
}

func TestGinOtel_ResponseRecording(t *testing.T) {
	testCases := []struct {
		name            string
		setupMiddleware func(*gin.Engine)
		setupHandler    func(*gin.Context)
		expectRecorded  bool
	}{
		{
			name: "default not record response",
			setupMiddleware: func(r *gin.Engine) {
				r.Use(GinOtel())
			},
			setupHandler: func(c *gin.Context) {
				c.String(http.StatusOK, "Hello, World!")
			},
			expectRecorded: false,
		},
		{
			name: "when set WithRecordResponseWhenLogrusError, if logrus error, record response",
			setupMiddleware: func(r *gin.Engine) {
				r.Use(GinOtel(WithRecordResponseWhenLogrusError(true)))
			},
			setupHandler: func(c *gin.Context) {
				logrus.WithContext(c.Request.Context()).Error("test")
				c.String(http.StatusOK, "Hello, World!")
			},
			expectRecorded: true,
		},
		{
			name: "when WithRecordResponse returns true, record response",
			setupMiddleware: func(r *gin.Engine) {
				r.Use(GinOtel(WithRecordResponse(func(c *gin.Context) bool {
					return true
				})))
			},
			setupHandler: func(c *gin.Context) {
				c.String(http.StatusOK, "Hello, World!")
			},
			expectRecorded: true,
		},
		{
			name: `when WithFilterRecordResponse returns true, not record response,
			  regardless of WithRecordResponseWhenLogrusError and WithRecordResponse`,
			setupMiddleware: func(r *gin.Engine) {
				r.Use(GinOtel(
					WithRecordResponseWhenLogrusError(true),
					WithRecordResponse(func(c *gin.Context) bool {
						return true
					}),
					WithFilterRecordResponse(func(c *gin.Context) bool {
						return true
					}),
				))
			},
			setupHandler: func(c *gin.Context) {
				logrus.WithContext(c.Request.Context()).Error("test")
				c.String(http.StatusOK, "Hello, World!")
			},
			expectRecorded: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// each test case setup a independent test environment
			exporter, tp, router := setupTracingTest()

			// setup middleware
			tc.setupMiddleware(router)

			// setup handler
			router.GET("/test", func(c *gin.Context) {
				tc.setupHandler(c)
			})

			// execute the request
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", http.NoBody)
			router.ServeHTTP(w, req)

			// get the recorded spans
			tp.ForceFlush(context.Background())
			spans := exporter.GetSpans().Snapshots()

			// verify the result
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.expectRecorded, hasResponseBodyAttribute(spans),
				"response body recording status does not match the expected result")

			// reset the exporter for the next test
			exporter.Reset()
		})
	}
}
