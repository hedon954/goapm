package goapm

import (
	"context"
	"time"

	"github.com/hedon954/goapm/goapm/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewAPM creates a new APM component, which is a wrapper of opentelemetry.
func NewAPM(otelEndpoint string) (closeFunc func(), err error) {
	ctx := context.Background()

	// setup a resource
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(
			internal.BuildInfo.AppName(),
		)),
	)
	if err != nil {
		return nil, err
	}

	// connect to otel collector
	conn, err := grpc.NewClient(otelEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	// setup a trace exporter
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		_ = traceProvider.Shutdown(context.Background())
	}, nil
}
