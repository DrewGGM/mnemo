// Package store owns the schema, its migrations and every SQL statement.
package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var embedded embed.FS

const migrationsDir = "migrations"

// migrateLockKey serializes concurrent migration runs, such as two replicas
// starting at once, on a session advisory lock.
const migrateLockKey int64 = 0x6d6e656d6f // "mnemo"

// statementTimeout caps every query of the service pool (quality-baseline,
// PostgreSQL row).
const statementTimeout = 5 * time.Second

var migrationName = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.sql$`)

type migration struct {
	version int
	file    string
}

// Migrate applies the embedded migrations that schema_migrations does not
// record yet, in version order and each in its own transaction. adminURL is
// the mnemo_admin connection: only the role that owns the schema can migrate it.
func Migrate(ctx context.Context, adminURL string) error {
	return migrate(ctx, adminURL, embedded)
}

func migrate(ctx context.Context, adminURL string, fsys fs.FS) error {
	ms, err := listMigrations(fsys)
	if err != nil {
		return err
	}
	// pgx quotes the connection string in parse errors and masks the password
	// only on a best-effort basis, so that error is replaced, never wrapped.
	connConfig, err := pgx.ParseConfig(adminURL)
	if err != nil {
		return errors.New("migrate: MNEMO_ADMIN_DATABASE_URL is not a valid connection string")
	}
	conn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return fmt.Errorf("migrate: connect: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {
		return fmt.Errorf("migrate: lock: %w", err)
	}
	if err := requirePgvector(ctx, conn); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    int PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("migrate: create schema_migrations: %w", err)
	}
	for _, m := range ms {
		if err := apply(ctx, conn, fsys, m); err != nil {
			return fmt.Errorf("migrate: %s: %w", m.file, err)
		}
	}
	return nil
}

func apply(ctx context.Context, conn *pgx.Conn, fsys fs.FS, m migration) error {
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		var applied bool
		err := tx.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", m.version).Scan(&applied)
		if err != nil || applied {
			return err
		}
		body, err := fs.ReadFile(fsys, path.Join(migrationsDir, m.file))
		if err != nil {
			return err
		}
		// Without arguments pgx uses the simple protocol, which runs every
		// statement of the file inside this transaction.
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.version)
		return err
	})
}

// listMigrations returns the migrations in version order. fs.ReadDir sorts
// by name, and the fixed four-digit prefix makes that the version order.
func listMigrations(fsys fs.FS) ([]migration, error) {
	entries, err := fs.ReadDir(fsys, migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("migrate: list migrations: %w", err)
	}
	ms := make([]migration, 0, len(entries))
	for _, e := range entries {
		match := migrationName.FindStringSubmatch(e.Name())
		if match == nil {
			return nil, fmt.Errorf("migrate: %s: name must be NNNN_name.sql", e.Name())
		}
		version, _ := strconv.Atoi(match[1]) // the pattern only admits four digits
		if n := len(ms); n > 0 && ms[n-1].version == version {
			return nil, fmt.Errorf("migrate: %s and %s share version %d", ms[n-1].file, e.Name(), version)
		}
		ms = append(ms, migration{version: version, file: e.Name()})
	}
	return ms, nil
}

// requirePgvector checks the installed pgvector, or the one the server would
// install, before anything is applied (assumption A-9).
func requirePgvector(ctx context.Context, conn *pgx.Conn) error {
	var version string
	err := conn.QueryRow(ctx, `SELECT coalesce(
		(SELECT extversion FROM pg_extension WHERE extname = 'vector'),
		(SELECT default_version FROM pg_available_extensions WHERE name = 'vector'),
		'')`).Scan(&version)
	if err != nil {
		return fmt.Errorf("migrate: read pgvector version: %w", err)
	}
	return checkPgvector(version)
}

func checkPgvector(version string) error {
	if version == "" {
		return errors.New("migrate: pgvector is not available in this PostgreSQL; use an image with pgvector >= 0.8")
	}
	majorText, rest, _ := strings.Cut(version, ".")
	minorText, _, _ := strings.Cut(rest, ".")
	major, errMajor := strconv.Atoi(majorText)
	minor, errMinor := strconv.Atoi(minorText)
	if errMajor != nil || errMinor != nil || major == 0 && minor < 8 {
		return fmt.Errorf("migrate: pgvector >= 0.8 is required, this PostgreSQL provides %q", version)
	}
	return nil
}

// OpenPool opens the service pool from MNEMO_DATABASE_URL, with
// statement_timeout set on every connection.
func OpenPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	// pgx quotes the connection string in parse errors and masks the password
	// only on a best-effort basis, so that error is replaced, never wrapped.
	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.New("MNEMO_DATABASE_URL is not a valid connection string")
	}
	poolConfig.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(statementTimeout.Milliseconds(), 10)
	return pgxpool.NewWithConfig(ctx, poolConfig)
}

// Ready reports whether the database answers and its schema includes the
// latest embedded migration. A newer schema is accepted: migrations are
// additive, so a binary rolled back after `mnemo migrate` keeps serving.
func Ready(ctx context.Context, pool *pgxpool.Pool) error {
	ms, err := listMigrations(embedded)
	if err != nil {
		return err
	}
	var current int
	if err := pool.QueryRow(ctx, "SELECT coalesce(max(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("ready: read schema version: %w", err)
	}
	if latest := ms[len(ms)-1].version; current < latest {
		return fmt.Errorf("ready: schema at version %d, want at least %d", current, latest)
	}
	return nil
}
