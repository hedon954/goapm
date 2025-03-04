package goapm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/cloudflare/tableflip"
	"github.com/gin-gonic/gin"
	redisv6 "github.com/go-redis/redis"
	"github.com/hedon954/goapm/apm"
	protos "github.com/hedon954/goapm/fixtures"
	"github.com/hedon954/goapm/internal/testutils"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestGoAPM_infra_smoke_should_success(t *testing.T) {
	mysqlDSN, mysqlShutdown := testutils.PrepareMySQL()
	defer mysqlShutdown()
	redisDSN, redisShutdown := testutils.PrepareRedis()
	defer redisShutdown()

	deferCalled := false
	prependDeferCalled := false
	closerCalled := false

	tmpDir, err := os.MkdirTemp("", "goapm-test-*")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	var infra *Infra
	assert.NotPanics(t, func() {
		infra = NewInfra("test",
			WithTableflip(tableflip.Options{
				UpgradeTimeout: 10 * time.Minute,
				PIDFile:        filepath.Join(tmpDir, "goapm-test.pid"),
			}, syscall.SIGUSR2),
			WithMySQL("mysql", mysqlDSN),
			WithGorm("gorm", mysqlDSN),
			WithRedisV6("redis", &redisv6.Options{
				Addr: redisDSN,
			}),
			WithRedisV9("redis9", &redis.Options{
				Addr: redisDSN,
			}),
			WithAPM("http://127.0.0.1:4317"),
			WithAutoPProf(&apm.AutoPProfOpt{
				EnableCPU:       true,
				EnableMem:       true,
				EnableGoroutine: true,
			}),
			WithMetrics(prometheus.NewCounter(prometheus.CounterOpts{
				Name: "test_counter",
				Help: "test counter",
			})),
			WithRotateLog(tmpDir, rotatelogs.WithRotationTime(time.Hour*24)),
			WithCloser(func() { closerCalled = true }),
		)
	})
	assert.NotNil(t, infra)
	infra.Defer(func() { deferCalled = true })
	infra.PrependDefer(func() { prependDeferCalled = true })

	// check redis and mysql and gorm
	assert.NotNil(t, infra.Gorm("gorm"))
	assert.NotNil(t, infra.MySQL("mysql"))
	assert.NotNil(t, infra.RedisV6("redis"))
	assert.NotNil(t, infra.RedisV9("redis9"))

	// check defer and closer
	assert.False(t, deferCalled)
	assert.False(t, prependDeferCalled)
	assert.False(t, closerCalled)

	// check gin
	testGin(infra, t)

	// check http server
	testHTTPServer(infra, t)

	// check grpc server and client
	testGRPCServerAndClient(infra, t)

	// stop infra
	go func() {
		time.Sleep(1 * time.Second)
		infra.upg.Stop()
	}()
	infra.WaitToStop()
	infra.Stop()

	// check defer and closer
	assert.True(t, deferCalled)
	assert.True(t, prependDeferCalled)
	assert.True(t, closerCalled)
}

func testGin(infra *Infra, t *testing.T) {
	// tableflip
	listener, err := infra.upg.Listen("tcp", ":")
	assert.Nil(t, err)
	infra.Defer(func() { assert.Nil(t, listener.Close()) })
	fmt.Printf("[%s]gin listen at: %s\n", infra.FullName(), listener.Addr().String())

	// run gin
	ginServer := infra.NewGin(apm.PrometheusBasicAuth("admin", "admin"))
	ginServer.GET("/gin", func(c *gin.Context) {
		c.String(http.StatusOK, "gin ok")
	})
	go func() {
		assert.NotPanics(t, func() {
			_ = ginServer.RunListener(listener)
		})
	}()

	time.Sleep(1 * time.Second)

	// check gin server
	resp, err := http.Get(fmt.Sprintf("http://%s/gin", listener.Addr().String()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "gin ok", string(body))
}

func testHTTPServer(infra *Infra, t *testing.T) {
	// http server
	httpServer := infra.NewHTTPServer(":")
	httpServer.Handle("/http", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("http ok"))
	}))
	httpServer.Start()
	infra.Defer(httpServer.Close)

	time.Sleep(1 * time.Second)

	// check http server
	fmt.Printf("[%s]http server listen at: %s\n", infra.FullName(), httpServer.Addr())
	resp, err := http.Get(fmt.Sprintf("http://%s/http", httpServer.Addr()))
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, "http ok", string(body))
}

func testGRPCServerAndClient(infra *Infra, t *testing.T) {
	server := infra.NewGRPCServer(":")
	protos.RegisterHelloServiceServer(server, &helloSvc{})
	server.Start()
	infra.Defer(server.Stop)

	time.Sleep(100 * time.Millisecond)

	client, err := apm.NewGrpcClient(server.Addr(), "test server")
	assert.Nil(t, err)
	res, err := protos.NewHelloServiceClient(client).SayHello(context.Background(),
		&protos.HelloRequest{Name: "World"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello, World", res.Message)
}

type helloSvc struct {
	protos.UnimplementedHelloServiceServer
}

func (s *helloSvc) SayHello(ctx context.Context, in *protos.HelloRequest) (*protos.HelloResponse, error) {
	return &protos.HelloResponse{Message: "Hello, " + in.Name}, nil
}
