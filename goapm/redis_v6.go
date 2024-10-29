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
	*redis.Client
	tracer trace.Tracer
}

// NewRedisV6 creates a new redis client with otel tracing enabled.
func NewRedisV6(addr, password string, db ...int) (*RedisV6, error) {
	dbNum := 0
	if len(db) > 0 {
		dbNum = db[0]
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		DB:       dbNum,
		Password: password,
	})

	if err := rdb.Ping().Err(); err != nil {
		return nil, err
	}

	return &RedisV6{
		Client: rdb,
		tracer: otel.Tracer(redisV6TracerName),
	}, nil
}

// WithContext wraps client with context and wraps process and process pipeline with otel tracing.
func (r *RedisV6) WithContext(ctx context.Context) *redis.Client {
	client := r.Client.WithContext(ctx)
	wrapProcess(r.tracer, client)
	wrapProcessPipeline(r.tracer, client)
	return client
}

func wrapProcess(tracer trace.Tracer, client *redis.Client) {
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			_, span := tracer.Start(client.Context(), "redis.v6.processCmd")
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

func wrapProcessPipeline(tracer trace.Tracer, client *redis.Client) {
	client.WrapProcessPipeline(func(oldProcess func([]redis.Cmder) error) func([]redis.Cmder) error {
		return func(cmds []redis.Cmder) error {
			_, span := tracer.Start(client.Context(), "redis.v6.processPipelineCmd")
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
