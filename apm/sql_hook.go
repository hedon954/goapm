package apm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/xwb1989/sqlparser"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey string

const (
	ctxBeginTime ctxKey = "sqldb.begin"

	mysqlTracerName string = "goapm/mysql"
)

var (
	slowSqlThreshold = 1 * time.Second
	longTxThreshold  = 3 * time.Second
)

// SetSlowSqlThreshold sets the threshold for a slow SQL query.
func SetSlowSqlThreshold(d time.Duration) {
	slowSqlThreshold = d
}

// SetLongTxThreshold sets the threshold for a long transaction.
func SetLongTxThreshold(d time.Duration) {
	longTxThreshold = d
}

// NewMySQL returns a new MySQL driver with hooks.
func NewMySQL(name, connectURL string) (*sql.DB, error) {
	driverName := fmt.Sprintf("%s-%s", "mysql-wrapper", uuid.NewString())
	sql.Register(driverName, wrap(&mysql.MySQLDriver{}, name, connectURL))

	db, err := sql.Open(driverName, connectURL)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	Logger.Info(context.TODO(), fmt.Sprintf("mysql sql.DB client[%s] connected", name), nil)
	return db, nil
}

func wrap(d driver.Driver, name, connectURL string) driver.Driver {
	tracer := otel.Tracer(mysqlTracerName)
	dsn, err := mysql.ParseDSN(connectURL)
	if err != nil {
		panic("invalid mysql connect url: " + err.Error())
	}
	return &Driver{d, Hooks{
		Before: func(ctx context.Context, query string, args ...any) (context.Context, error) {
			// trace
			ctx = context.WithValue(ctx, ctxBeginTime, time.Now())
			if ctx, span := tracer.Start(ctx, "sqltrace"); span != nil {
				span.SetAttributes(
					attribute.String("mysql.name", name),
					attribute.String("sql", truncate(query)),
					attribute.String("args", truncate(sliceToString(args))),
				)
				return ctx, nil
			}
			return ctx, nil
		},
		After: func(ctx context.Context, query string, args ...any) (context.Context, error) {
			// metric
			table, op, multiTable, err := SQLParser.parseTable(query)
			if !multiTable && err == nil {
				libraryCounter.WithLabelValues(LibraryTypeMySQL, sqlparser.StmtType(op), table, dsn.DBName+"."+dsn.Addr).Inc()
			}

			// trace
			beginTime := time.Now()
			if begin := ctx.Value(ctxBeginTime); begin != nil {
				beginTime = begin.(time.Time)
			}
			span := trace.SpanFromContext(ctx)
			defer span.End()
			elapsed := time.Since(beginTime)
			if elapsed > slowSqlThreshold {
				span.SetAttributes(
					attribute.Bool("slowsql", true),
					attribute.Int64("sql_duration_ms", elapsed.Milliseconds()),
				)
			}

			// log
			switch op {
			case sqlparser.StmtInsert, sqlparser.StmtUpdate, sqlparser.StmtDelete:
				Logger.Info(ctx, "auditsql", map[string]any{
					"query":       query,
					"args":        args,
					"duration_ms": elapsed.Milliseconds(),
				})
			}
			return ctx, nil
		},
		OnError: func(ctx context.Context, err error, query string, args ...any) error {
			// trace
			span := trace.SpanFromContext(ctx)
			defer span.End()
			if !errors.Is(err, driver.ErrSkip) {
				span.SetAttributes(attribute.Bool("error", true))
				span.RecordError(err, trace.WithStackTrace(true), trace.WithTimestamp(time.Now()))
				return err
			}
			span.SetAttributes(attribute.Bool("drop", true))
			return err
		},
	}}
}

func truncate(query string) string {
	const maxLength = 1024
	if len(query) > maxLength {
		return query[:maxLength]
	}
	return query
}

func sliceToString(args []any) string {
	if len(args) == 0 {
		return "[]"
	}
	return fmt.Sprintf("%v", args)
}
