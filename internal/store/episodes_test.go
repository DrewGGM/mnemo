//go:build integration

package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// devTenant is the tenant that 0001_init.sql inserts.
var devTenant = [16]byte{15: 1}

// devPool opens a pool of mnemo_app on a freshly migrated database of the
// test's own, so the test does not depend on who migrated the shared one.
func devPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := newTestDB(t)
	db.migrate(t)
	pool, err := OpenPool(t.Context(), db.appURL)
	if err != nil {
		t.Fatalf("OpenPool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func devScope(t *testing.T) Scope {
	t.Helper()
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return Scope{TenantID: devTenant, SubjectID: "test-" + hex.EncodeToString(b)}
}

func insertOne(t *testing.T, pool *pgxpool.Pool, s Scope, content string) [16]byte {
	t.Helper()
	vec := make([]float32, 1024)
	vec[0] = 1
	ids, err := InsertEpisodes(t.Context(), pool, s, "s-1", []NewEpisode{{
		Role: "user", Content: content, Trust: "user",
		OccurredAt: time.Date(2026, 3, 10, 15, 4, 0, 0, time.UTC), Embedding: vec,
	}})
	if err != nil {
		t.Fatalf("InsertEpisodes: %v", err)
	}
	return ids[0]
}

func TestEpisodes_InsertStoresEmbedding(t *testing.T) {
	pool := devPool(t)
	s := devScope(t)
	id := insertOne(t, pool, s, "I moved to Medellin")
	var dims int
	err := WithScope(t.Context(), pool, s, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(),
			"SELECT vector_dims(embedding) FROM episodes WHERE tenant_id = $1 AND id = $2", s.TenantID, id).Scan(&dims)
	})
	if err != nil {
		t.Fatalf("read embedding: %v", err)
	}
	if dims != 1024 {
		t.Errorf("stored embedding has %d dimensions, want 1024", dims)
	}
}

func TestEpisodes_InsertStoresFts(t *testing.T) {
	pool := devPool(t)
	s := devScope(t)
	id := insertOne(t, pool, s, "I moved to Medellin")
	var matches bool
	err := WithScope(t.Context(), pool, s, func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(),
			"SELECT fts @@ plainto_tsquery('english', 'moving') FROM episodes WHERE tenant_id = $1 AND id = $2",
			s.TenantID, id).Scan(&matches)
	})
	if err != nil {
		t.Fatalf("read fts: %v", err)
	}
	if !matches {
		t.Error("the stored tsvector does not match a stemmed word of the content")
	}
}

// singleConnPool hands out one connection, so consecutive uses of the pool
// share it.
func singleConnPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(testURL("MNEMO_TEST_DATABASE_URL", "mnemo_app"))
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestWithScope_SetsTenantForTheTransaction(t *testing.T) {
	pool := singleConnPool(t)
	var inside, after string
	err := WithScope(t.Context(), pool, devScope(t), func(tx pgx.Tx) error {
		return tx.QueryRow(t.Context(), "SELECT current_setting('app.tenant_id')").Scan(&inside)
	})
	if err != nil {
		t.Fatalf("WithScope: %v", err)
	}
	if want := "00000000-0000-0000-0000-000000000001"; inside != want {
		t.Errorf("app.tenant_id inside the transaction = %q, want %q", inside, want)
	}
	err = pool.QueryRow(t.Context(), "SELECT coalesce(current_setting('app.tenant_id', true), '')").Scan(&after)
	if err != nil {
		t.Fatalf("read app.tenant_id after the transaction: %v", err)
	}
	if after != "" {
		t.Errorf("app.tenant_id after the transaction = %q, want it unset on the pooled connection", after)
	}
}

func TestWithScope_AcquireTimesOut(t *testing.T) {
	pool := singleConnPool(t)
	held, err := pool.Acquire(t.Context())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer held.Release()

	err = WithScope(t.Context(), pool, devScope(t), func(pgx.Tx) error { return nil })
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, ErrUnavailable) {
		t.Errorf("WithScope on an exhausted pool = %v, want a deadline error wrapped in ErrUnavailable", err)
	}
}

func TestWithScope_StatementTimeoutIsUnavailable(t *testing.T) {
	err := WithScope(t.Context(), singleConnPool(t), devScope(t), func(tx pgx.Tx) error {
		if _, err := tx.Exec(t.Context(), "SET LOCAL statement_timeout = '50ms'"); err != nil {
			return err
		}
		_, err := tx.Exec(t.Context(), "SELECT pg_sleep(5)")
		return err
	})
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("WithScope after statement_timeout = %v, want ErrUnavailable", err)
	}
}

func TestWithScope_StatementErrorIsNotUnavailable(t *testing.T) {
	err := WithScope(t.Context(), singleConnPool(t), devScope(t), func(tx pgx.Tx) error {
		_, err := tx.Exec(t.Context(), "SELECT 1/0")
		return err
	})
	if err == nil || errors.Is(err, ErrUnavailable) {
		t.Errorf("WithScope after a division by zero = %v, want an error that is not ErrUnavailable", err)
	}
}
