package debug

import (
	"context"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"fmt"
	"github.com/gofrs/uuid"
	"time"
)

const strMaxLen = 102400

type SkipDbLogging struct{}

// DebugDriver is a driver that logs all driver operations.
type DebugDriver struct {
	dialect.Driver                               // underlying driver.
	log            func(context.Context, ...any) // log function. defaults to log.Println.
}

// DebugWithContext gets a driver and a logging function, and returns
// a new debugged-driver that prints all outgoing operations with context.
func DebugWithContext(d dialect.Driver, logger func(context.Context, ...any)) dialect.Driver {
	drv := &DebugDriver{d, logger}
	return drv
}

// Exec logs its params and calls the underlying driver Exec method.
func (d *DebugDriver) Exec(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Driver.Exec(ctx, query, args, v)
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return err
	}

	d.log(ctx, fmt.Sprintf("driver.Exec: query=%v args=%v time=%v", query, args, time.Since(start)))
	return err
}

// ExecContext logs its params and calls the underlying driver ExecContext method if it is supported.
func (d *DebugDriver) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	drv, ok := d.Driver.(interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.ExecContext is not supported")
	}
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return drv.ExecContext(ctx, query, args...)
	}
	d.log(ctx, fmt.Sprintf("driver.ExecContext: query=%v args=%v", query, args))
	return drv.ExecContext(ctx, query, args...)
}

// Query logs its params and calls the underlying driver Query method.
func (d *DebugDriver) Query(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Driver.Query(ctx, query, args, v)
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return err
	}
	d.log(ctx, fmt.Sprintf("driver.Query: query=%v args=%v time=%v", query, args, time.Since(start)))
	return err
}

// QueryContext logs its params and calls the underlying driver QueryContext method if it is supported.
func (d *DebugDriver) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	drv, ok := d.Driver.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.QueryContext is not supported")
	}
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return drv.QueryContext(ctx, query, args...)
	}
	d.log(ctx, fmt.Sprintf("driver.QueryContext: query=%v args=%v", query, args))
	return drv.QueryContext(ctx, query, args...)
}

// Tx adds an log-id for the transaction and calls the underlying driver Tx command.
func (d *DebugDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := d.Driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	id := uuid.Must(uuid.NewV4()).String()
	d.log(ctx, fmt.Sprintf("driver.Tx(%s): started", id))
	return &DebugTx{tx, id, d.log, ctx}, nil
}

// BeginTx adds an log-id for the transaction and calls the underlying driver BeginTx command if it is supported.
func (d *DebugDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	drv, ok := d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.BeginTx is not supported")
	}
	tx, err := drv.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	id := uuid.Must(uuid.NewV4()).String()
	d.log(ctx, fmt.Sprintf("driver.BeginTx(%s): started", id))
	return &DebugTx{tx, id, d.log, ctx}, nil
}

// DebugTx is a transaction implementation that logs all transaction operations.
type DebugTx struct {
	dialect.Tx                               // underlying transaction.
	id         string                        // transaction logging id.
	log        func(context.Context, ...any) // log function. defaults to fmt.Println.
	ctx        context.Context               // underlying transaction context.
}

// Exec logs its params and calls the underlying transaction Exec method.
func (d *DebugTx) Exec(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Tx.Exec(ctx, query, args, v)
	printArgs := args
	if argsArray, ok := args.([]interface{}); ok {
		for i, argVal := range argsArray {
			if argValStr, ok := argVal.(string); ok && len(argValStr) > strMaxLen {
				printArgs.([]interface{})[i] = argValStr[:strMaxLen] + "...[Truncated]..."
			}
		}
	}
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return err
	}
	d.log(ctx, fmt.Sprintf("Tx(%s).Exec: query=%v args=%v time=%v", d.id, query, args, time.Since(start)))
	return err
}

// ExecContext logs its params and calls the underlying transaction ExecContext method if it is supported.
func (d *DebugTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	drv, ok := d.Tx.(interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Tx.ExecContext is not supported")
	}
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return drv.ExecContext(ctx, query, args...)
	}
	d.log(ctx, fmt.Sprintf("Tx(%s).ExecContext: query=%v args=%v", d.id, query, args))
	return drv.ExecContext(ctx, query, args...)
}

// Query logs its params and calls the underlying transaction Query method.
func (d *DebugTx) Query(ctx context.Context, query string, args, v any) error {
	start := time.Now()
	err := d.Tx.Query(ctx, query, args, v)
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return err
	}
	d.log(ctx, fmt.Sprintf("Tx(%s).Query: query=%v args=%v time=%v", d.id, query, args, time.Since(start)))
	return err
}

// QueryContext logs its params and calls the underlying transaction QueryContext method if it is supported.
func (d *DebugTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	drv, ok := d.Tx.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Tx.QueryContext is not supported")
	}
	if skip, ok := ctx.Value(SkipDbLogging{}).(bool); ok && skip {
		return drv.QueryContext(ctx, query, args...)
	}
	d.log(ctx, fmt.Sprintf("Tx(%s).QueryContext: query=%v args=%v", d.id, query, args))
	return drv.QueryContext(ctx, query, args...)
}

// Commit logs this step and calls the underlying transaction Commit method.
func (d *DebugTx) Commit() error {
	d.log(d.ctx, fmt.Sprintf("Tx(%s): committed", d.id))
	return d.Tx.Commit()
}

// Rollback logs this step and calls the underlying transaction Rollback method.
func (d *DebugTx) Rollback() error {
	d.log(d.ctx, fmt.Sprintf("Tx(%s): rollbacked", d.id))
	return d.Tx.Rollback()
}
