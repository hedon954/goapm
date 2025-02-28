package apm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

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
func hasAttributes(spans []sdktrace.ReadOnlySpan, attrs []string) bool {
	if len(spans) == 0 {
		return false
	}

	for _, expectedAttr := range attrs {
		hasExpectedAttr := false
		for _, span := range spans {
			for _, attr := range span.Attributes() {
				if string(attr.Key) == expectedAttr {
					hasExpectedAttr = true
					break
				}
			}
			if hasExpectedAttr {
				break
			}
		}
		if !hasExpectedAttr {
			return false
		}
	}
	return true
}

// getAttributeValue gets the attribute value from the spans
func getAttributeValue(spans []sdktrace.ReadOnlySpan, expectedAttr string) string {
	for _, span := range spans {
		for _, attr := range span.Attributes() {
			if string(attr.Key) == expectedAttr {
				return string(attr.Value.AsString())
			}
		}
	}
	return ""
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
			assert.Equal(t, tc.expectRecorded, hasAttributes(spans, []string{"http.response.body"}),
				"response body recording status does not match the expected result")

			// reset the exporter for the next test
			exporter.Reset()
		})
	}
}

func TestGinOtel_WithPanic_should_call_hooks(t *testing.T) {
	hook1Called := false
	panicHook1 := func(c *gin.Context, panic any, stack []byte) (stop bool) {
		hook1Called = true
		return false
	}

	hook2Called := false
	panicHook2 := func(c *gin.Context, panic any, stack []byte) (stop bool) {
		hook2Called = true
		return false
	}

	router := gin.Default()
	// setup middleware
	router.Use(GinOtel(WithPanicHook(panicHook1), WithPanicHook(panicHook2)))

	// setup handler
	router.POST("/panic", func(c *gin.Context) {
		panic("panic")
	})

	// execute the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/panic", http.NoBody)
	router.ServeHTTP(w, req)

	// verify the result
	assert.True(t, hook1Called)
	assert.True(t, hook2Called)
}

func TestGinOtel_WithPanic_should_record_query_and_form_params(t *testing.T) {
	// each test case setup a independent test environment
	exporter, tp, router := setupTracingTest()
	defer exporter.Reset()

	// setup handler
	router.Use(GinOtel())
	router.POST("/form", func(c *gin.Context) {
		var req struct {
			Username string `form:"username"`
			Password string `form:"password"`
			Age      int    `form:"age"`
		}
		_ = c.ShouldBind(&req)
		panic(req)
	})

	// execute the request
	w := httptest.NewRecorder()

	form := url.Values{}
	form.Add("username", "testuser")
	form.Add("password", "password123")
	form.Add("age", "25")
	req, _ := http.NewRequest("POST", "/form?a=1&b=2&c=string", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	// get the recorded spans
	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans().Snapshots()

	// verify the result
	assert.True(t, hasAttributes(spans, []string{
		"http.request.params",
		"http.request.path",
		"http.request.method",
		"http.request.query",
		"http.response.code",
		"error",
	}))

	assert.Equal(t, getAttributeValue(spans, "http.request.method"), "POST")
	assert.Equal(t, getAttributeValue(spans, "http.request.path"), "/form")
	assert.Contains(t, getAttributeValue(spans, "http.request.query"), "string")
	assert.Contains(t, getAttributeValue(spans, "http.request.params"), "password123")
}

func TestGinOtel_WithPanic_should_record_query_and_json_body(t *testing.T) {
	// each test case setup a independent test environment
	exporter, tp, router := setupTracingTest()
	defer exporter.Reset()

	router.Use(GinOtel(WithRecordResponse(func(c *gin.Context) bool {
		return true
	})))

	// setup handler
	router.POST("/json", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Age      int    `json:"age"`
		}
		_ = c.ShouldBind(&req)
		panic(req)
	})

	// execute the request
	w := httptest.NewRecorder()
	body := `{"username": "testuser", "password": "password123", "age": 25}`
	req, _ := http.NewRequest("POST", "/json?a=1&b=2&c=string", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// get the recorded spans
	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans().Snapshots()

	// verify the result
	assert.True(t, hasAttributes(spans, []string{
		"http.request.body",
		"http.request.path",
		"http.request.method",
		"http.request.query",
		"http.response.code",
		"error",
	}))

	assert.Equal(t, getAttributeValue(spans, "http.request.method"), "POST")
	assert.Equal(t, getAttributeValue(spans, "http.request.path"), "/json")
	assert.Contains(t, getAttributeValue(spans, "http.request.query"), "string")
	assert.Contains(t, getAttributeValue(spans, "http.request.body"), "password123")
}

func TestGinOtel_business_error_code_msg_should_be_recorded(t *testing.T) {
	// each test case setup a independent test environment
	exporter, tp, router := setupTracingTest()
	defer exporter.Reset()

	router.Use(GinOtel())
	router.POST("/error", func(c *gin.Context) {
		c.Header(HeaderBusinessErrorCode, "123456")
		c.Header(HeaderBusinessErrorMsg, "business error")
		c.String(http.StatusOK, "Hello, World!")
	})

	// execute the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/error", http.NoBody)
	router.ServeHTTP(w, req)

	// get the recorded spans
	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans().Snapshots()

	// verify the result
	assert.True(t, hasAttributes(spans, []string{
		"http.response.business_error_code",
		"http.response.business_error_msg",
	}))
}

func TestGinOtel_responseFormat(t *testing.T) {
	// each test case setup a independent test environment
	exporter, tp, router := setupTracingTest()
	defer exporter.Reset()

	router.Use(GinOtel(WithRecordResponse(func(c *gin.Context) bool {
		return true
	}), WithResponseFormat(func(c *gin.Context, body *bytes.Buffer) string {
		return "response after format"
	})))

	router.POST("/format", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"code": "123456", "msg": "business error"})
	})

	// execute the request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/format", http.NoBody)
	router.ServeHTTP(w, req)

	// get the recorded spans
	tp.ForceFlush(context.Background())
	spans := exporter.GetSpans().Snapshots()

	// verify the result
	assert.True(t, hasAttributes(spans, []string{
		"http.response.body",
	}))
	assert.Equal(t, getAttributeValue(spans, "http.response.body"), "response after format")
}
