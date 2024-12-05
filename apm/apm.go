package apm

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/encoding/gzip"

	"github.com/hedon954/goapm/internal"
)

type apmBuilder struct {
	res       *resource.Resource
	grpcToken string
	sampler   sdktrace.Sampler
}

// ApmOption is the option for the apm.
type ApmOption func(b *apmBuilder)

// WithResource sets the resource for the apm, if not set, a default resource will be created.
func WithResource(res *resource.Resource) ApmOption {
	return func(b *apmBuilder) {
		b.res = res
	}
}

// WithGRPCAuthToken sets the grpc auth token for the apm, it is optional.
func WithGRPCAuthToken(token string) ApmOption {
	return func(b *apmBuilder) {
		b.grpcToken = token
	}
}

// WithSampler sets the custom sampler for the apm, it is optional.
func WithSampler(sampler sdktrace.Sampler) ApmOption {
	return func(b *apmBuilder) {
		b.sampler = sampler
	}
}

// NewAPM creates a new APM component, which is a wrapper of opentelemetry.
func NewAPM(otelEndpoint string, opts ...ApmOption) (closeFunc func(), err error) {
	ctx := context.Background()

	b := &apmBuilder{}
	for _, opt := range opts {
		opt(b)
	}

	if b.sampler == nil {
		b.sampler = sdktrace.AlwaysSample()
	}

	if b.res == nil {
		// setup a resource
		res, err := resource.New(ctx,
			resource.WithHost(),
			resource.WithProcess(),
			resource.WithTelemetrySDK(),
			resource.WithAttributes(semconv.ServiceName(
				internal.BuildInfo.AppName(),
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create otel resource: %w", err)
		}
		b.res = res
	}

	// setup a trace exporter
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"Authentication": b.grpcToken,
		}),
		otlptracegrpc.WithCompressor(gzip.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel trace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(b.sampler),
		sdktrace.WithResource(b.res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(traceProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := traceProvider.Shutdown(ctx); err != nil {
			otel.Handle(err)
		}
	}, nil
}
