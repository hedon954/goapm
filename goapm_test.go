package goapm

import (
	"os"
	"testing"
	"time"

	"github.com/cloudflare/tableflip"
	redisv6 "github.com/go-redis/redis"
	"github.com/hedon954/goapm/apm"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestGoAPM_infra_smoke_should_success(t *testing.T) {
	deferCalled := false
	closerCalled := false

	tmpDir, err := os.MkdirTemp("", "goapm-test-*")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	var infra *Infra
	assert.NotPanics(t, func() {
		infra = NewInfra("test",
			WithTableflip(tableflip.Options{}),
			WithMySQL("mysql", "root:root@tcp(127.0.0.1:3306)/goapm"),
			WithGorm("gorm", "root:root@tcp(127.0.0.1:3306)/goapm"),
			WithRedisV6("redis", &redisv6.Options{
				Addr: "127.0.0.1:6379",
			}),
			WithRedisV9("redis9", &redis.Options{
				Addr: "127.0.0.1:6379",
			}),
			WithAPM("http://127.0.0.1:4317"),
			WithAutoPProf(&apm.AutoPProfOpt{
				EnableCPU:       true,
				EnableMem:       true,
				EnableGoroutine: true,
			}),
			WithMetrics(nil),
			WithRotateLog(tmpDir, rotatelogs.WithRotationTime(time.Hour*24)),
			WithCloser(func() { closerCalled = true }),
		)
	})
	assert.NotNil(t, infra)
	infra.Defer(func() { deferCalled = true })

	// check redis and mysql and gorm
	assert.NotNil(t, infra.Gorm("gorm"))
	assert.NotNil(t, infra.MySQL("mysql"))
	assert.NotNil(t, infra.RedisV6("redis"))
	assert.NotNil(t, infra.RedisV9("redis9"))

	// check defer and closer
	assert.False(t, deferCalled)
	assert.False(t, closerCalled)

	// stop infra
	infra.Stop()

	// check defer and closer
	assert.True(t, deferCalled)
	assert.True(t, closerCalled)
}
