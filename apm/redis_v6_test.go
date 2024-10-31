package apm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisV6(t *testing.T) {
	client, err := NewRedisV6("test", "127.0.0.1:6379", "", 10)
	assert.Nil(t, err)
	defer client.Close()

	_, err = client.WithContext(context.Background()).Set("haha", "world", 0).Result()
	assert.Nil(t, err)

	res, err := client.WithContext(context.Background()).Get("haha").Result()
	assert.Nil(t, err)
	assert.Equal(t, "world", res)
}
