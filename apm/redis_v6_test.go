package apm

import (
	"context"
	"testing"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

func TestRedisV6(t *testing.T) {
	client, err := NewRedisV6("test", &redis.Options{
		Addr: redisDSN,
		DB:   10,
	})
	assert.Nil(t, err)
	defer client.Close()

	_, err = client.WithContext(context.Background()).Set("haha", "world", 0).Result()
	assert.Nil(t, err)

	res, err := client.WithContext(context.Background()).Get("haha").Result()
	assert.Nil(t, err)
	assert.Equal(t, "world", res)
}
