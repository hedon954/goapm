package goapm

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/prometheus/client_golang/prometheus"
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
type Infra struct {
	// Tracer is the tracer for the infra,
	Tracer trace.Tracer
	Name   string

	redisV6s map[string]*apm.RedisV6
	redisV9s map[string]*redis.Client
	mysqls   map[string]*sql.DB
	gorms    map[string]*gorm.DB

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

// WithMySQL creates a new mysql db and adds it to the infra.
// name is the business name of the db, and addr is the address of the db.
func WithMySQL(name, addr string) InfraOption {
	return func(infra *Infra) {
		if infra.mysqls[name] != nil {
			panic(fmt.Errorf("goapm mysql db already exists: %s", name))
		}
		db, err := apm.NewMySQL(name, addr)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm mysql db: %w", err))
		}
		infra.mysqls[name] = db
		infra.closeFuncs = append(infra.closeFuncs, func() {
			_ = db.Close()
			apm.Logger.Info(context.TODO(), "goapm mysql db closed", map[string]any{"name": name})
		})
	}
}

// WithGorm creates a new gorm db and adds it to the infra.
// name is the business name of the db, and addr is the address of the db.
func WithGorm(name, addr string) InfraOption {
	return func(infra *Infra) {
		if infra.mysqls[name] != nil {
			panic(fmt.Errorf("goapm gorm db already exists: %s", name))
		}
		db, err := apm.NewGorm(name, addr)
		if err != nil {
			panic(fmt.Errorf("failed to create goapm gorm db: %w", err))
		}
		infra.gorms[name] = db
		infra.closeFuncs = append(infra.closeFuncs, func() {
			d, _ := db.DB()
			if d != nil {
				_ = d.Close()
				apm.Logger.Info(context.TODO(), "goapm gorm db closed", map[string]any{"name": name})
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
			panic(fmt.Errorf("failed to create goapm redis v6 client: %w", err))
		}
		infra.redisV6s[name] = client
		infra.closeFuncs = append(infra.closeFuncs, func() {
			_ = client.Close()
			apm.Logger.Info(context.TODO(), "goapm redis v6 client closed", map[string]any{"name": name})
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
			panic(fmt.Errorf("failed to create goapm redis v9 client: %w", err))
		}
		infra.redisV9s[name] = client
		infra.closeFuncs = append(infra.closeFuncs, func() {
			_ = client.Close()
			apm.Logger.Info(context.TODO(), "goapm redis v9 client closed", map[string]any{"name": name})
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

// Stop stops the infra.
func (infra *Infra) Stop() {
	for _, fn := range infra.closeFuncs {
		fn()
	}

	apm.Logger.Info(context.TODO(), "goapm infra finished stopping", map[string]any{
		"name": infra.Name,
	})
}
