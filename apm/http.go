package apm

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	httpTracerName = "goapm/httpServer"

	HeaderBusinessErrorCode = "X-Business-Error-Code"
	HeaderBusinessErrorMsg  = "X-Business-Error-Msg"
)

// HTTPServer is a wrapper around http.Server that adds tracing to the server.
type HTTPServer struct {
	mux *http.ServeMux
	*http.Server
	tracer   trace.Tracer
	listener net.Listener
}

// NewHTTPServer creates a new HTTPServer,
// it is a wrapper around http.Server that adds tracing and metrics to the server.
func NewHTTPServer(addr string) *HTTPServer {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("failed to listen goapm http server: %w", err))
	}

	return NewHTTPServer2(listener)
}

// NewHTTPServer2 creates a new HTTPServer with a given listener,
// it is a wrapper around http.Server that adds tracing and metrics to the server.
func NewHTTPServer2(listener net.Listener) *HTTPServer {
	mux := http.NewServeMux()
	srv := &HTTPServer{
		tracer: otel.Tracer(httpTracerName),
		mux:    mux,
		Server: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 30 * time.Second, //nolint:mnd
		},
		listener: listener,
	}

	srv.Handle("/metrics", promhttp.HandlerFor(MetricsReg, promhttp.HandlerOpts{
		Registry: MetricsReg,
	}))
	srv.Handle("/heartbeat", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	return srv
}

// Addr returns the address of the http server.
func (s *HTTPServer) Addr() string {
	return s.listener.Addr().String()
}

// Start starts the http server in a new goroutine.
func (s *HTTPServer) Start() {
	go func() {
		Logger.Info(context.Background(), "starting http server", nil)
		err := s.Server.Serve(s.listener)
		if err != nil && err != http.ErrServerClosed {
			Logger.Error(context.Background(), "failed to start http server", err, nil)
		}
	}()
}

// Close shutdowns the http server.
func (s *HTTPServer) Close() {
	if s.Server != nil {
		if err := s.Server.Shutdown(context.Background()); err != nil {
			Logger.Error(context.Background(), "failed to shutdown http server", err, nil)
		}
	}
}

// Handle registers a new handler for the given pattern.
func (s *HTTPServer) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, &traceHandler{
		handler: handler,
		tracer:  s.tracer,
	})
}

// HandleFunc registers a new handler function for the given pattern.
func (s *HTTPServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.Handle(pattern, &traceHandler{
		handler: http.HandlerFunc(handler),
		tracer:  s.tracer,
	})
}

// traceHandler is a wrapper around http.Handler that adds tracing to the handler.
type traceHandler struct {
	handler http.Handler
	tracer  trace.Tracer
}

func (th *traceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if th.handler == nil {
		th.handler.ServeHTTP(w, r)
		return
	}

	// metrics
	ServerHandleCounter.WithLabelValues(MetricTypeHTTP, r.Method+"."+r.URL.Path, "", "").Inc()

	// trace
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
	ctx, span := th.tracer.Start(ctx, "HTTP "+r.Method+" "+r.URL.Path)
	defer span.End()
	r = r.Clone(ctx)
	respWrapper := &responseWrapper{ResponseWriter: w}

	start := time.Now()
	func() {
		// panic recover
		defer func() {
			if err := recover(); err != nil {
				span.SetAttributes(attribute.Bool("error", true))
				span.RecordError(
					fmt.Errorf("%v", err),
					trace.WithStackTrace(true),
					trace.WithTimestamp(time.Now()),
				)

				// log
				Logger.Error(ctx, "panic in http handler", fmt.Errorf("panic: %v", err), map[string]any{
					"method": r.Method,
					"path":   r.URL.Path,
					"params": r.Form.Encode(),
					"stack":  string(debug.Stack()),
				})
				http.Error(respWrapper, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		// handle request
		th.handler.ServeHTTP(respWrapper, r)
	}()

	// http response status code
	if respWrapper.status == 0 {
		respWrapper.status = http.StatusOK
	}
	elapsed := time.Since(start)
	span.SetAttributes(
		attribute.Int("http.response.code", respWrapper.status),
		attribute.Int64("http.duration_ms", elapsed.Milliseconds()),
	)

	// business error code
	// TODO: check if needs
	businessErrorCode := respWrapper.Header().Get(HeaderBusinessErrorCode)
	businessErrorMsg := respWrapper.Header().Get(HeaderBusinessErrorMsg)
	if businessErrorCode != "" {
		span.SetAttributes(
			attribute.String("http.response.business_error_code", businessErrorCode),
			attribute.String("http.response.business_error_msg", businessErrorMsg),
		)
	}

	// metrics
	ServerHandleHistogram.WithLabelValues(
		MetricTypeHTTP, r.Method+"."+r.URL.Path, strconv.Itoa(respWrapper.status), "", "",
	).Observe(elapsed.Seconds())
}

// responseWrapper is a wrapper around http.ResponseWriter that store the status code.
type responseWrapper struct {
	http.ResponseWriter
	status int
}

func (r *responseWrapper) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
