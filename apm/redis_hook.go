package apm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	redisTracerName = "goapm/redisV9"
)

// NewRedisV9 creates a new redis client with tracing.
// name is the business name of the redis client, it will be used in the span name.
func NewRedisV9(name string, opts *redis.Options) (*redis.Client, error) {
	client := redis.NewClient(opts)
	client.AddHook(&redisHook{name})

	res, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	if res != "PONG" {
		return nil, fmt.Errorf("redis ping failed: %s", res)
	}

	Logger.Info(context.TODO(), fmt.Sprintf("redis v9 client[%s] connected", name), nil)
	return client, nil
}

type redisHook struct {
	name string
}

func (h *redisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *redisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	tracer := otel.Tracer(redisTracerName)
	return func(ctx context.Context, cmd redis.Cmder) error {
		ctx, span := tracer.Start(ctx, fmt.Sprintf("redis.v9.processCmd-[%s]", h.name))
		defer span.End()

		span.SetAttributes(attribute.String("cmd", truncate(cmd.String())))

		err := next(ctx, cmd)
		if err != nil && !errors.Is(err, redis.Nil) {
			span.SetAttributes(attribute.Bool("error", true))
			span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
		}
		return err
	}
}

func (h *redisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	tracer := otel.Tracer(redisTracerName)
	return func(ctx context.Context, cmds []redis.Cmder) error {
		ctx, span := tracer.Start(ctx, fmt.Sprintf("redis.v9.processPipelineCmd-[%s]", h.name))
		defer span.End()

		span.SetAttributes(attribute.String("cmd", truncate(fmt.Sprintf("%v", cmds))))

		err := next(ctx, cmds)
		if err != nil && !errors.Is(err, redis.Nil) {
			span.SetAttributes(attribute.Bool("error", true))
			span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
		}
		return err
	}
}
