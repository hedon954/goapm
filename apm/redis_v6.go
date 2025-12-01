package apm

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RedisV6 is a wrapper of redis.Client with otel tracing enabled.
type RedisV6 struct {
	name string
	*redis.Client
}

// NewRedisV6 creates a new redis client with otel tracing enabled.
// name is the business name of the redis client, it will be used in the span name.
func NewRedisV6(name string, opts *redis.Options) (*RedisV6, error) {
	rdb := redis.NewClient(opts)

	if err := rdb.Ping().Err(); err != nil {
		return nil, err
	}

	Logger.Info(context.TODO(), fmt.Sprintf("redis v6 client[%s] connected", name), nil)
	return &RedisV6{
		name:   name,
		Client: rdb,
	}, nil
}

// WithContext wraps client with context and wraps process and process pipeline with otel tracing.
func (r *RedisV6) WithContext(ctx context.Context) *redis.Client {
	client := r.Client.WithContext(ctx)
	wrapProcess(r.name, client)
	wrapProcessPipeline(r.name, client)
	return client
}

func wrapProcess(name string, client *redis.Client) {
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			span := trace.SpanFromContext(client.Context())
			if span == nil || !span.IsRecording() {
				return oldProcess(cmd)
			}

			eventOpt := trace.WithAttributes(attribute.String("cmd", cmdStr(cmd)))
			err := oldProcess(cmd)
			if err != nil {
				eventOpt = trace.WithAttributes(
					attribute.String("cmd", truncate(cmdStr(cmd))),
					attribute.String("error_msg", err.Error()),
				)
				span.SetAttributes(attribute.Bool("error", true))
				span.SetStatus(codes.Error, err.Error())
				CustomerRecordError(span, err, true, 5)
			}
			span.AddEvent(fmt.Sprintf("redis.v6.processCmd-[%s]", name), eventOpt)
			return err
		}
	})
}

func wrapProcessPipeline(name string, client *redis.Client) {
	client.WrapProcessPipeline(func(oldProcess func([]redis.Cmder) error) func([]redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			span := trace.SpanFromContext(client.Context())
			if span == nil || !span.IsRecording() {
				return oldProcess(cmds)
			}

			eventOpt := trace.WithAttributes(attribute.String("cmd", cmdStr(cmds...)))
			err := oldProcess(cmds)
			if err != nil {
				eventOpt = trace.WithAttributes(
					attribute.String("cmd", truncate(cmdStr(cmds...))),
					attribute.String("error_msg", err.Error()),
				)
				span.SetAttributes(attribute.Bool("error", true))
				span.SetStatus(codes.Error, err.Error())
				CustomerRecordError(span, err, true, 5)
			}
			span.AddEvent(fmt.Sprintf("redis.v6.processPipelineCmd-[%s]", name), eventOpt)
			return err
		}
	})
}

func cmdStr(cmds ...redis.Cmder) string {
	var cmdStr string
	for i, cmd := range cmds {
		cmdStr += fmt.Sprintf("%s %v", cmd.Name(), cmd.Args())
		if i != len(cmds)-1 {
			cmdStr += "\n"
		}
	}
	return truncate(cmdStr)
}
