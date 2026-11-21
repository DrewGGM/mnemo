package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// acquireTimeout bounds the wait for a pooled connection, so an exhausted
// pool fails the request instead of queueing it without end (quality-baseline,
// PostgreSQL row; TH-11).
const acquireTimeout = 2 * time.Second

// ErrUnavailable wraps the errors of WithScope that come from the database
// being out of reach or out of time, not from the statement: no connection,
// an exhausted pool, a deadline or statement_timeout (quality-baseline,
// PostgreSQL row).
var ErrUnavailable = errors.New("database unavailable")

// Scope is the tenant and subject that a use case reads and writes.
type Scope struct {
	TenantID  [16]byte
	SubjectID string
}

// WithScope runs fn in a transaction with app.tenant_id set to the scope's
// tenant for that transaction only. It commits when fn returns nil and rolls
// back otherwise.
func WithScope(ctx context.Context, pool *pgxpool.Pool, s Scope, fn func(pgx.Tx) error) error {
	err := withScope(ctx, pool, s, fn)
	if err != nil && unavailable(err) {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return err
}

func withScope(ctx context.Context, pool *pgxpool.Pool, s Scope, fn func(pgx.Tx) error) error {
	acquireCtx, cancel := context.WithTimeout(ctx, acquireTimeout)
	conn, err := pool.Acquire(acquireCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1::uuid::text, true)", s.TenantID); err != nil {
			return fmt.Errorf("set tenant: %w", err)
		}
		return fn(tx)
	})
}

// unavailable reports whether err means the database could not serve the
// request in time, as opposed to rejecting it.
func unavailable(err error) bool {
	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) || errors.Is(err, context.DeadlineExceeded) || pgconn.Timeout(err) {
		return true
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case "57014", // query_canceled, raised by statement_timeout
		"57P01", "57P02", "57P03", // admin_shutdown, crash_shutdown, cannot_connect_now
		"53300": // too_many_connections
		return true
	}
	return strings.HasPrefix(pgErr.Code, "08") // connection_exception
}
