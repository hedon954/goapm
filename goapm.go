package goapm

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudflare/tableflip"
	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"mosn.io/holmes"

	// import this package to fix the issue: https://github.com/open-telemetry/opentelemetry-collector/issues/10476
	// since we need to specify the version of google.golang.org/genproto, but we do not use it in the code,
	// so we need to import it to avoid deleting it by the go mod tidy command
	_ "google.golang.org/genproto/protobuf/api"

	"github.com/hedon954/goapm/apm"
)

// Infra is an infrastructure manager for goapm.
// It is recommended to create a single instance of Infra and share it across the application.
// TODO: add a print function to print the infra's components and closers.
type Infra struct {
	// Name is the business name of the infra.
	Name string
	// Tracer is the tracer for the infra,
	Tracer trace.Tracer
	// Upgrader is the tableflip for the infra,
	upg *tableflip.Upgrader

	redisV6s map[string]*apm.RedisV6
	redisV9s map[string]*redis.Client
	mysqls   map[string]*sql.DB
	gorms    map[string]*gorm.DB

	// closeFuncs holds the functions to close the infra.
	// It should be closed in the reverse order of the creation.
	closeFuncs []func()
}

// InfraOption is the option for Infra.
type InfraOption func(*Infra)

// NewInfra creates a new infra with the given options.
func NewInfra(name string, opts ...InfraOption) *Infra {
	infra := &Infra{
		Name:       name,
		Tracer:     otel.Tracer(fmt.Sprintf("goapm/service/%s", name)),
		redisV6s:   make(map[string]*apm.RedisV6),
		redisV9s:   make(map[string]*redis.Client),
		mysqls:     make(map[string]*sql.DB),
		gorms:      make(map[string]*gorm.DB),
		closeFuncs: make([]func(), 0),
	}
	for _, opt := range opts {
		opt(infra)
	}
	return infra
}

// WithTableflip creates a new tableflip and adds it to the infra.
// The tableflip is used to support graceful restart.
// If the tableflip is created, the infra will listen the ports with it for http and rpc servers.
// NOTE: we recommend that this should be the first option to be called.
func WithTableflip(opts tableflip.Options, sigs ...os.Signal) InfraOption {
	// default signal is SIGUSR2
	if len(sigs) == 0 {
		sigs = []os.Signal{syscall.SIGUSR2}
	}

	upg, err := tableflip.New(opts)
	if err != nil {
		panic(fmt.Errorf("failed to create goapm tableflip: %w", err))
	}

	// listen the SIGUSR2 signal to trigger the process restart
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, sigs...)
		for s := range sig {
			if err := upg.Upgrade(); err != nil {
				apm.Logger.Error(context.TODO(), "goapm tableflip upgrade failed", err, map[string]any{
					"signal": s.String(),
				})
			}
		}
	}()

	return func(infra *Infra) {
		infra.upg = upg
		infra.closeFuncs = append([]func(){
			func() {
				upg.Stop()
				apm.Logger.Info(context.TODO(), "goapm tableflip stopped", map[string]any{"name": infra.Name})
			},
		}, infra.closeFuncs...) // tableflip should be the last one to be closed
	}
}

// WithMySQL creates a new mysql db and adds it to the infra.
// name is the business name of the db, and addr is the address of the db.
func WithMySQL(name, addr string) InfraOption {
	return func(infra *Infra) {
		if infra.mysqls[name] != nil {
			panic(fmt.Errorf("goapm mysql db already exists: %s", name))
		}
		db, err := apm.NewMySQL(name, addr)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm mysql db[%s]: %w", name, err))
		}
		infra.mysqls[name] = db
		infra.closeFuncs = append(infra.closeFuncs, func() {
			_ = db.Close()
			apm.Logger.Info(context.TODO(), "goapm mysql db[%s] closed", map[string]any{"name": name})
		})
	}
}

// WithGorm creates a new gorm db and adds it to the infra.
// name is the business name of the db, and addr is the address of the db.
func WithGorm(name, addr string) InfraOption {
	return func(infra *Infra) {
		if infra.gorms[name] != nil {
			panic(fmt.Errorf("goapm gorm db already exists: %s", name))
		}
		db, err := apm.NewGorm(name, addr)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm gorm db[%s]: %w", name, err))
		}
		infra.gorms[name] = db
		infra.closeFuncs = append(infra.closeFuncs, func() {
			d, _ := db.DB()
			if d != nil {
				_ = d.Close()
				apm.Logger.Info(context.TODO(), "goapm gorm db[%s] closed", map[string]any{"name": name})
			}
		})
	}
}

// WithRedisV6 creates a new redis v6 client and adds it to the infra.
// name is the business name of the redis, and addr is the address of the redis.
func WithRedisV6(name, addr, password string, db ...int) InfraOption {
	return func(infra *Infra) {
		if infra.redisV6s[name] != nil {
			panic(fmt.Errorf("goapm redis v6 client already exists: %s", name))
		}
		client, err := apm.NewRedisV6(name, addr, password, db...)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm redis v6 client[%s]: %w", name, err))
		}
		infra.redisV6s[name] = client
		infra.closeFuncs = append(infra.closeFuncs, func() {
			_ = client.Close()
			apm.Logger.Info(context.TODO(), "goapm redis v6 client[%s] closed", map[string]any{"name": name})
		})
	}
}

// WithRedisV9 creates a new redis v9 client and adds it to the infra.
// name is the business name of the redis, and addr is the address of the redis.
func WithRedisV9(name, addr, password string, db ...int) InfraOption {
	return func(infra *Infra) {
		if infra.redisV9s[name] != nil {
			panic(fmt.Errorf("goapm redis v9 client already exists: %s", name))
		}
		client, err := apm.NewRedisV9(name, addr, password, db...)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm redis v9 client[%s]: %w", name, err))
		}
		infra.redisV9s[name] = client
		infra.closeFuncs = append(infra.closeFuncs, func() {
			_ = client.Close()
			apm.Logger.Info(context.TODO(), "goapm redis v9 client[%s] closed", map[string]any{"name": name})
		})
	}
}

// WithMetrics registers the given collectors to the goapm metrics registry.
// It default provides some collectors defined in goapm/metric.go.
func WithMetrics(collectors ...prometheus.Collector) InfraOption {
	return func(infra *Infra) {
		apm.MetricsReg.MustRegister(collectors...)
	}
}

// WithAutoPProf starts a holmes dumper to automatically record the running state of the program.
func WithAutoPProf(autoPProfOpts *apm.AutoPProfOpt, opts ...holmes.Option) InfraOption {
	return func(infra *Infra) {
		h, err := apm.NewHomes(autoPProfOpts, opts...)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm homes: %w", err))
		}
		h.Start()
		apm.Logger.Info(context.TODO(), "auto pprof started", map[string]any{
			"enable_cpu":       autoPProfOpts.EnableCPU,
			"enable_mem":       autoPProfOpts.EnableMem,
			"enable_goroutine": autoPProfOpts.EnableGoroutine,
		})
		infra.closeFuncs = append(infra.closeFuncs, func() {
			h.Stop()
			apm.Logger.Info(context.TODO(), "auto pprof stopped", nil)
		})
	}
}

// WithAPM creates a new apm and adds it to the infra.
func WithAPM(otelEndpoint string) InfraOption {
	return func(infra *Infra) {
		closeFunc, err := apm.NewAPM(otelEndpoint)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm apm: %w", err))
		}
		infra.closeFuncs = append(infra.closeFuncs, closeFunc)
	}
}

// WithRotateLog creates a new rotate log and sets it to the logrus.
// It default rotates every 7 days and keeps 7 days' logs.
func WithRotateLog(path string, opts ...rotatelogs.Option) InfraOption {
	defaultOpts := []rotatelogs.Option{
		rotatelogs.WithRotationTime(time.Hour * 24 * 7),
		rotatelogs.WithRotationCount(24 * 7),
	}

	return func(infra *Infra) {
		writer, err := rotatelogs.New(path, append(defaultOpts, opts...)...)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm rotate log: %w", err))
		}
		logrus.SetOutput(writer)
	}
}

// WithCloser adds a closer to the infra.
func WithCloser(fn func()) InfraOption {
	return func(infra *Infra) {
		infra.closeFuncs = append(infra.closeFuncs, fn)
	}
}

// MySQL returns the mysql db client with the given name.
func (infra *Infra) MySQL(name string) *sql.DB {
	return infra.mysqls[name]
}

// Gorm returns the gorm db client with the given name.
func (infra *Infra) Gorm(name string) *gorm.DB {
	return infra.gorms[name]
}

// RedisV6 returns the redis v6 client with the given name.
func (infra *Infra) RedisV6(name string) *apm.RedisV6 {
	return infra.redisV6s[name]
}

// RedisV9 returns the redis v9 client with the given name.
func (infra *Infra) RedisV9(name string) *redis.Client {
	return infra.redisV9s[name]
}

// AppendCloser appends a closer to the infra.
func (infra *Infra) AppendCloser(fn func()) {
	infra.closeFuncs = append(infra.closeFuncs, fn)
}

// PrependCloser prepends a closer to the infra.
func (infra *Infra) PrependCloser(fn func()) {
	infra.closeFuncs = append([]func(){fn}, infra.closeFuncs...)
}

// NewHTTPServer creates a new http server with the given address.
// If the tableflip is created, the server will listen on the address with the tableflip.
// Otherwise, it will listen on the address directly.
func (infra *Infra) NewHTTPServer(addr string) *apm.HTTPServer {
	if infra.upg == nil {
		return apm.NewHTTPServer(addr)
	}
	listener, err := infra.upg.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("failed to listen goapm http server with tableflip: %w", err))
	}
	return apm.NewHTTPServer2(listener)
}

// NewGin creates a new gin engine with otel tracing and metrics.
// It will automatically add the otel tracing and metrics middleware to the engine.
// If metricsAuth is not nil, it will add a metrics handler with the given auth middleware.
func (infra *Infra) NewGin(metricsAuth gin.HandlerFunc, opts ...gin.OptionFunc) *gin.Engine {
	res := gin.New(opts...)
	res.Use(apm.GinOtel())

	metricsHandler := gin.WrapH(
		promhttp.HandlerFor(
			apm.MetricsReg,
			promhttp.HandlerOpts{Registry: apm.MetricsReg},
		),
	)

	if metricsAuth != nil {
		res.GET("/metrics", metricsAuth, metricsHandler)
	} else {
		res.GET("/metrics", metricsHandler)
	}

	return res
}

// NewGRPCServer creates a new grpc server with the given address.
// If the tableflip is created, the server will listen on the address with the tableflip.
func (infra *Infra) NewGRPCServer(addr string) *apm.GrpcServer {
	if infra.upg == nil {
		return apm.NewGrpcServer(addr)
	}
	listener, err := infra.upg.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Errorf("failed to listen goapm grpc server with tableflip: %w", err))
	}
	return apm.NewGrpcServer2(listener)
}

// Tableflip returns the tableflip of the infra.
func (infra *Infra) Tableflip() *tableflip.Upgrader {
	return infra.upg
}

// Stop stops the infra.
func (infra *Infra) Stop() {
	// close the components in the reverse order of the creation
	for i := len(infra.closeFuncs) - 1; i >= 0; i-- {
		infra.closeFuncs[i]()
	}

	apm.Logger.Info(context.TODO(), "goapm infra finished stopping", map[string]any{
		"name": infra.Name,
	})
}

// WaitToStop waits for the infra to stop.
// It should be called in front of the infra.Stop().
func (infra *Infra) WaitToStop() {
	if upg := infra.upg; upg != nil {
		// when the new process starts successfully,
		// calling upg.Ready will clear invalid fds and send a signal
		// to the parent process indicating that initialization is complete.
		if err := upg.Ready(); err != nil {
			apm.Logger.Error(context.TODO(), "goapm tableflip ready failed", err, map[string]any{"name": infra.Name})
		} else {
			apm.Logger.Info(context.TODO(), "goapm tableflip ready success", map[string]any{"name": infra.Name})
		}
		<-upg.Exit()
	}
}
