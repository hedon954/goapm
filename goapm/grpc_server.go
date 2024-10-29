package goapm

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hedon954/goapm/goapm/internal"
)

const (
	grpcServerTracerName = "goapm/grpcServer"
)

// GrpcServer is a wrapper of grpc.Server.
type GrpcServer struct {
	addr string
	*grpc.Server
}

func NewGrpcServer(addr string, opts ...grpc.ServerOption) *GrpcServer {
	options := []grpc.ServerOption{
		grpc.UnaryInterceptor(unaryServerInterceptor()),
	}
	options = append(options, opts...)

	server := grpc.NewServer(options...)
	return &GrpcServer{
		addr:   addr,
		Server: server,
	}
}

func (s *GrpcServer) Start() {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		panic("GRPC server listen failed: " + err.Error())
	}
	go func() {
		log.Printf("[%s][%s] starting grpc server on: %s\n", internal.BuildInfo.AppName(), internal.BuildInfo.Hostname(), s.addr)
		if err := s.Server.Serve(listener); err != nil {
			panic("GRPC server serve failed: " + err.Error())
		}
	}()
}

func (s *GrpcServer) Stop() {
	s.Server.GracefulStop()
}

func unaryServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(grpcServerTracerName)

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
			serverHandleHistogram.WithLabelValues(
				MetricTypeGRPC, info.FullMethod, statusCode.String(), peerApp, peerHost,
			).Observe(time.Since(start).Seconds())
		}()

		// metric
		serverHandleCounter.WithLabelValues(MetricTypeGRPC, info.FullMethod, peerApp, peerHost).Inc()

		// call the handler
		resp, err := handler(ctx, req)

		// set the status and error on the span
		if err != nil {
			s, ok := status.FromError(err)
			if ok {
				statusCode = s.Code()
			}
			span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
			span.SetAttributes(attribute.Bool("haserror", true))
			span.SetAttributes(attribute.String("grpc.status_code", s.Code().String()))
		}

		return resp, err
	}
}
