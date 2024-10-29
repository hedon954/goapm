package goapm

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	redisV6TracerName = "goapm/redisV6"
)

// RedisV6 is a wrapper of redis.Client with otel tracing enabled.
type RedisV6 struct {
	name string
	*redis.Client
	tracer trace.Tracer
}

// NewRedisV6 creates a new redis client with otel tracing enabled.
// name is the business name of the redis client, it will be used in the span name.
func NewRedisV6(name, addr, password string, db ...int) (*RedisV6, error) {
	dbIndex := 0
	if len(db) > 0 {
		dbIndex = db[0]
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       dbIndex,
		Password: password,
	})

	if err := rdb.Ping().Err(); err != nil {
		return nil, err
	}

	return &RedisV6{
		name:   name,
		Client: rdb,
		tracer: otel.Tracer(redisV6TracerName),
	}, nil
}

// WithContext wraps client with context and wraps process and process pipeline with otel tracing.
func (r *RedisV6) WithContext(ctx context.Context) *redis.Client {
	client := r.Client.WithContext(ctx)
	wrapProcess(r.tracer, r.name, client)
	wrapProcessPipeline(r.tracer, r.name, client)
	return client
}

func wrapProcess(tracer trace.Tracer, name string, client *redis.Client) {
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			_, span := tracer.Start(client.Context(), fmt.Sprintf("redis.v6.processCmd-[%s]", name))
			defer span.End()

			span.SetAttributes(attribute.String("cmd", cmdStr(cmd)))

			err := oldProcess(cmd)
			if err != nil {
				span.SetAttributes(attribute.Bool("haserror", true))
				span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
			}
			return err
		}
	})
}

func wrapProcessPipeline(tracer trace.Tracer, name string, client *redis.Client) {
	client.WrapProcessPipeline(func(oldProcess func([]redis.Cmder) error) func([]redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			_, span := tracer.Start(client.Context(), fmt.Sprintf("redis.v6.processPipelineCmd-[%s]", name))
			defer span.End()

			span.SetAttributes(attribute.String("cmd", cmdStr(cmds...)))

			err := oldProcess(cmds)
			if err != nil {
				span.SetAttributes(attribute.Bool("haserror", true))
				span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
			}
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
