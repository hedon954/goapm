package apm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	return func(ctx context.Context, cmd redis.Cmder) error {
		if ctx == nil || cmd == nil || cmd.Args() == nil {
			return next(ctx, cmd)
		}
		span := trace.SpanFromContext(ctx)
		if span == nil || !span.IsRecording() {
			return next(ctx, cmd)
		}

		eventOpt := trace.WithAttributes(attribute.String("cmd", trimArgs(cmd.Args())))
		err := next(ctx, cmd)
		if err != nil && !errors.Is(err, redis.Nil) {
			eventOpt = trace.WithAttributes(
				attribute.String("cmd", truncate(cmd.String())),
				attribute.String("error_msg", err.Error()),
			)
			span.SetAttributes(attribute.Bool("error", true))
			span.SetStatus(codes.Error, err.Error())
			CustomerRecordError(span, err, true, 5)
		}
		span.AddEvent(fmt.Sprintf("redis.v9.processCmd-[%s]", h.name), eventOpt)
		return err
	}
}

func (h *redisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		span := trace.SpanFromContext(ctx)
		if span == nil || !span.IsRecording() {
			return next(ctx, cmds)
		}

		eventOpt := trace.WithAttributes(attribute.String("cmd", truncate(fmt.Sprintf("%v", cmds))))
		err := next(ctx, cmds)
		if err != nil && !errors.Is(err, redis.Nil) {
			eventOpt = trace.WithAttributes(
				attribute.String("cmd", truncate(fmt.Sprintf("%v", cmds))),
				attribute.String("error_msg", err.Error()),
			)
			span.SetAttributes(attribute.Bool("error", true))
			span.SetStatus(codes.Error, err.Error())
			CustomerRecordError(span, err, true, 5)
		}
		span.AddEvent(fmt.Sprintf("redis.v9.processPipelineCmd-[%s]", h.name), eventOpt)
		return err
	}
}

func trimArgs(args []interface{}) string {
	res := fmt.Sprintf("%v", args)
	res = strings.TrimPrefix(res, "[")
	res = strings.TrimSuffix(res, "]")
	return res
}
