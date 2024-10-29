package goapm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedisHook(t *testing.T) {
	client, err := NewRedisV9("test", "127.0.0.1:6379", "")
	assert.Nil(t, err)
	defer client.Close()

	_, err = client.Set(context.Background(), "haha", "world", 0).Result()
	assert.Nil(t, err)

	res, err := client.Get(context.Background(), "haha").Result()
	assert.Nil(t, err)
	assert.Equal(t, "world", res)
}
