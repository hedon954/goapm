package apm

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/hedon954/goapm/apm/internal"
)

const (
	grpcClientTracerName = "goapm/grpcClient"
)

// GrpcClient is a wrapper around grpc.ClientConn that provides tracing, metrics, and logging.
type GrpcClient struct {
	*grpc.ClientConn
}

func NewGrpcClient(addr, server string, opts ...grpc.DialOption) (*GrpcClient, error) {
	options := []grpc.DialOption{
		grpc.WithUnaryInterceptor(unaryClientInterceptor(server)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	options = append(options, opts...)

	conn, err := grpc.NewClient(addr, options...)
	if err != nil {
		return nil, err
	}
	return &GrpcClient{conn}, nil
}

func unaryClientInterceptor(server string) grpc.UnaryClientInterceptor {
	tracer := otel.Tracer(grpcClientTracerName)

	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// trace
		ctx, span := tracer.Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
		start := time.Now()
		defer func() {
			span.SetAttributes(attribute.Int64("grpc.duration_ms", time.Since(start).Milliseconds()))
			span.End()

			// metric
			clientHandleHistogram.WithLabelValues(MetricTypeGRPC, method, server).Observe(time.Since(start).Seconds())
		}()

		// set peer info into metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		md.Set(metadataKeyPeerApp, internal.BuildInfo.AppName())
		md.Set(metadataKeyPeerHost, internal.BuildInfo.Hostname())
		otel.GetTextMapPropagator().Inject(ctx, &metadataSupplier{metadata: &md})
		ctx = metadata.NewOutgoingContext(ctx, md)

		// metric
		clientHandleCounter.WithLabelValues(MetricTypeGRPC, method, server).Inc()

		// invoke the actual call
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
			span.SetAttributes(attribute.Bool("haserror", true))
			s, ok := status.FromError(err)
			if ok {
				span.SetAttributes(attribute.String("grpc.status_code", s.Code().String()))
			}
		}
		return err
	}
}
