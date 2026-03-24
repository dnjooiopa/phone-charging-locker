package dbctx

import (
	"context"
	"database/sql"
)

type ctxKey struct{}

// NewContext returns a new context with the given database.
func NewContext(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, ctxKey{}, db)
}

func from(ctx context.Context) *sql.DB {
	return ctx.Value(ctxKey{}).(*sql.DB)
}

// Query executes a query that returns rows.
func Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return from(ctx).QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row.
func QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return from(ctx).QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning any rows.
func Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return from(ctx).ExecContext(ctx, query, args...)
}
