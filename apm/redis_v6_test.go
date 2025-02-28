package apm

import (
	"context"
	"testing"

	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
)

func TestRedisV6(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
		return
	}

	client, err := NewRedisV6("test", &redis.Options{
		Addr: "127.0.0.1:6379",
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
