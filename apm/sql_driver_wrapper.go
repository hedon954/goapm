package apm

import (
	"context"
	"database/sql/driver"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Hooks is a set of hooks that can be invoked during the execution of a SQL query.
type Hooks struct {
	// Before is called before the query is executed.
	Before func(ctx context.Context, query string, args ...any) (context.Context, error)
	// After is called after the query is executed.
	After func(ctx context.Context, query string, args ...any) (context.Context, error)
	// OnError is called if the query fails.
	OnError func(ctx context.Context, err error, query string, args ...any) error
}

// Driver is a wrapper around the driver.Driver interface.
type Driver struct {
	driver.Driver
	hooks Hooks
}

// Open returns a new connection to the database.
// The name is a string in a driver-specific format.
//
// Open may return a cached connection (one previously
// closed), but doing so is unnecessary; the sql package
// maintains a pool of idle connections for efficient re-use.
//
// The returned connection is only used by one goroutine at a
// time.
func (d *Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}

	return &Conn{
		Conn:  conn,
		hooks: d.hooks,
	}, nil
}

// Conn is a wrapper around the driver.Conn interface.
// In order to hook into the execution of SQL queries,
// it should implement the following interfaces:
// - driver.ExecerContext
// - driver.QueryerContext
// - driver.ConnPrepareContext
type Conn struct {
	driver.Conn
	hooks Hooks
}

//nolint:dupl
func (conn *Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	var err error

	list := namedToAny(args)

	if ctx, err = conn.hooks.Before(ctx, query, list...); err != nil {
		return nil, err
	}

	results, err := conn.execContext(ctx, query, args)
	if err != nil {
		return results, conn.hooks.OnError(ctx, err, query, list...)
	}

	if _, err := conn.hooks.After(ctx, query, list...); err != nil {
		return results, err
	}

	return results, nil
}

func (conn *Conn) execContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	switch c := conn.Conn.(type) {
	case driver.ExecerContext:
		return c.ExecContext(ctx, query, args)
	default:
		return nil, fmt.Errorf("Conn does not implement driver.ExecerContext, got %T", c)
	}
}

//nolint:dupl
func (conn *Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	var err error

	list := namedToAny(args)

	if ctx, err = conn.hooks.Before(ctx, query, list...); err != nil {
		return nil, err
	}

	rows, err := conn.queryContext(ctx, query, args)
	if err != nil {
		return rows, conn.hooks.OnError(ctx, err, query, list...)
	}

	if _, err := conn.hooks.After(ctx, query, list...); err != nil {
		return rows, err
	}

	return rows, nil
}

func (conn *Conn) queryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch c := conn.Conn.(type) {
	case driver.QueryerContext:
		return c.QueryContext(ctx, query, args)
	default:
		return nil, fmt.Errorf("Conn does not implement driver.QueryerContext, got %T", c)
	}
}

// PrepareContext returns a prepared statement, bound to this connection.
// context is for the preparation of the statement,
// it must not store the context within the statement itself.
func (conn *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	var (
		stmt driver.Stmt
		err  error
	)

	if c, ok := conn.Conn.(driver.ConnPrepareContext); ok {
		stmt, err = c.PrepareContext(ctx, query)
	} else {
		err = fmt.Errorf("Conn does not implement driver.ConnPrepareContext, got %T", conn.Conn)
	}

	if err != nil {
		return nil, err
	}
	return &Stmt{stmt, conn.hooks, query}, nil
}

// Stmt is a wrapper around the driver.Stmt interface.
// In order to hook into the execution of SQL queries after preparation,
// it should implement the following interfaces:
// - driver.StmtExecContext
// - driver.StmtQueryContext
type Stmt struct {
	driver.Stmt
	hooks Hooks
	query string
}

// ExecContext executes a query that doesn't return rows, such
// as an INSERT or UPDATE.
//
// ExecContext must honor the context timeout and return when it is canceled.
//
//nolint:dupl
func (s *Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	var err error

	list := namedToAny(args)

	if ctx, err = s.hooks.Before(ctx, s.query, list...); err != nil {
		return nil, err
	}

	results, err := s.execContext(ctx, args)
	if err != nil {
		return results, s.hooks.OnError(ctx, err, s.query, list...)
	}

	if _, err := s.hooks.After(ctx, s.query, list...); err != nil {
		return results, err
	}

	return results, nil
}

func (s *Stmt) execContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	switch c := s.Stmt.(type) {
	case driver.StmtExecContext:
		return c.ExecContext(ctx, args)
	default:
		return nil, fmt.Errorf("Stmt does not implement driver.StmtExecContext, got %T", s.Stmt)
	}
}

// QueryContext executes a query that may return rows, such as a
// SELECT.
//
// QueryContext must honor the context timeout and return when it is canceled.
//
//nolint:dupl
func (s *Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	var err error

	list := namedToAny(args)

	if ctx, err = s.hooks.Before(ctx, s.query, list...); err != nil {
		return nil, err
	}

	rows, err := s.queryContext(ctx, args)
	if err != nil {
		return rows, s.hooks.OnError(ctx, err, s.query, list...)
	}

	if _, err := s.hooks.After(ctx, s.query, list...); err != nil {
		return rows, err
	}
	return rows, nil
}

func (s *Stmt) queryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	switch c := s.Stmt.(type) {
	case driver.StmtQueryContext:
		return c.QueryContext(ctx, args)
	default:
		return nil, fmt.Errorf("Stmt does not implement driver.StmtQueryContext, got %T", s.Stmt)
	}
}

// DriverTx is a wrapper around the driver.Tx interface.
// It should implement the following interfaces:
// - driver.Tx
// And the wrapped Conn need to implement driver.ConnBeginTx interface.
type DriverTx struct {
	driver.Tx
	start           time.Time
	ctx             context.Context
	longTxThreshold time.Duration
}

// BeginTx starts and returns a new transaction.
// If the context is canceled by the user the sql package will
// call Tx.Rollback before discarding and closing the connection.
//
// This must check opts.Isolation to determine if there is a set
// isolation level. If the driver does not support a non-default
// level and one is set or if there is a non-default isolation level
// that is not supported, an error must be returned.
//
// This must also check opts.ReadOnly to determine if the read-only
// value is true to either set the read-only transaction property if supported
// or return an error if it is not supported.
func (conn *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	tx, err := conn.beginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &DriverTx{tx, time.Now(), ctx, longTxThreshold}, nil
}

func (conn *Conn) beginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	switch c := conn.Conn.(type) {
	case driver.ConnBeginTx:
		return c.BeginTx(ctx, opts)
	default:
		return nil, fmt.Errorf("Conn does not implement driver.ConnBeginTx, got %T", conn.Conn)
	}
}

func (dt *DriverTx) Commit() error {
	err := dt.Tx.Commit()
	elapsed := time.Since(dt.start)
	if elapsed >= dt.longTxThreshold {
		if span := trace.SpanFromContext(dt.ctx); span != nil {
			span.SetAttributes(
				attribute.Bool("longtx", true),
				attribute.Int64("tx_duration_ms", elapsed.Milliseconds()),
			)
		}
	}
	return err
}

func (dt *DriverTx) Rollback() error {
	err := dt.Tx.Rollback()
	elapsed := time.Since(dt.start)
	if elapsed >= dt.longTxThreshold {
		if span := trace.SpanFromContext(dt.ctx); span != nil {
			span.SetAttributes(
				attribute.Bool("longtx", true),
				attribute.Int64("tx_duration_ms", elapsed.Milliseconds()),
			)
		}
	}
	return err
}

func namedToAny(args []driver.NamedValue) []any {
	res := make([]any, len(args))
	for i, arg := range args {
		res[i] = arg.Value
	}
	return res
}
