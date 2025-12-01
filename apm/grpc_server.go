package apm

import (
	"context"
	"fmt"
	"net"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hedon954/goapm/internal"
)

const (
	grpcServerTracerName = "goapm/grpcServer-"
)

// GrpcServer is a wrapper of grpc.Server.
type GrpcServer struct {
	*grpc.Server
	listener net.Listener
}

// NewGrpcServer creates a new grpc server with the given address.
func NewGrpcServer(addr string, opts ...grpc.ServerOption) *GrpcServer {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("failed to listen goapm rpc server: %w", err))
	}
	return NewGrpcServer2(listener, opts...)
}

// NewGrpcServer2 creates a new grpc server with the given listener.
func NewGrpcServer2(listener net.Listener, opts ...grpc.ServerOption) *GrpcServer {
	options := []grpc.ServerOption{
		UnaryInterceptor(unaryServerInterceptor()),
	}
	options = append(options, opts...)

	server := grpc.NewServer(options...)
	return &GrpcServer{
		listener: listener,
		Server:   server,
	}
}

// UnaryInterceptor is used to replace `grpc.UnaryInterceptor()`,
// in order to combine the goapm interceptor with the user-defined interceptors.
//
// PANIC: user should not use grpc.UnaryInterceptor() directly!!!!
func UnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(interceptors...)
}

func (s *GrpcServer) Addr() string {
	return s.listener.Addr().String()
}

func (s *GrpcServer) Start() {
	go func() {
		Logger.Info(context.Background(),
			fmt.Sprintf("[%s][%s] starting grpc server on: %s",
				internal.BuildInfo.AppName(),
				internal.BuildInfo.Hostname(),
				s.listener.Addr().String(),
			), nil,
		)
		if err := s.Server.Serve(s.listener); err != nil {
			panic("GRPC server serve failed: " + err.Error())
		}
	}()
}

func (s *GrpcServer) Stop() {
	s.Server.GracefulStop()
}

func unaryServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(grpcServerTracerName + internal.BuildInfo.AppName())

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// get the metadata from the incoming context or create a new one
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		peerApp, peerHost := getPeerInfo(md)

		// extract the metadata from the context
		ctx = otel.GetTextMapPropagator().Extract(ctx, &metadataSupplier{metadata: &md})

		// trace: start the span
		ctx, span := tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))

		statusCode := codes.OK
		start := time.Now()
		defer func() {
			span.SetAttributes(attribute.String("grpc.duration_ms", fmt.Sprintf("%d", time.Since(start).Milliseconds())))
			span.End()

			// metric
			ServerHandleHistogram.WithLabelValues(
				MetricTypeGRPC, info.FullMethod, statusCode.String(), peerApp, peerHost,
			).Observe(time.Since(start).Seconds())
		}()

		// metric
		ServerHandleCounter.WithLabelValues(MetricTypeGRPC, info.FullMethod, peerApp, peerHost).Inc()

		// call the handler
		resp, err := handler(ctx, req)

		// set the status and error on the span
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				statusCode = s.Code()
			}
			CustomerRecordError(span, err, true, 5)
			span.SetAttributes(attribute.Bool("error", true))
			span.SetStatus(otelcodes.Error, err.Error())
			span.SetAttributes(attribute.String("grpc.status_code", s.Code().String()))
		}

		return resp, err
	}
}
