//go:build integration

package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testURL returns the URL in env, or the development default of
// docker-compose.yml for role.
func testURL(env, role string) string {
	if u := os.Getenv(env); u != "" {
		return u
	}
	port := os.Getenv("MNEMO_DB_PORT")
	if port == "" {
		port = "55432"
	}
	return "postgres://" + role + ":" + role + "_dev@127.0.0.1:" + port + "/mnemo?sslmode=disable"
}

// testDB is an empty database created for one test and dropped after it, so
// migrations always start from nothing.
type testDB struct {
	adminURL, appURL string
}

func newTestDB(t *testing.T) testDB {
	t.Helper()
	adminURL := testURL("MNEMO_TEST_ADMIN_DATABASE_URL", "mnemo_admin")
	suffix := make([]byte, 8)
	_, _ = rand.Read(suffix)
	name := pgx.Identifier{"mnemo_t_" + hex.EncodeToString(suffix)}.Sanitize()

	conn, err := pgx.Connect(t.Context(), adminURL)
	if err != nil {
		t.Fatalf("connect as mnemo_admin (is `docker compose up -d --wait db` running?): %v", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	if _, err := conn.Exec(t.Context(), "CREATE DATABASE "+name); err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		c, err := pgx.Connect(context.Background(), adminURL)
		if err != nil {
			t.Errorf("drop test database: %v", err)
			return
		}
		defer func() { _ = c.Close(context.Background()) }()
		if _, err := c.Exec(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
	dbName := strings.Trim(name, `"`)
	return testDB{
		adminURL: withDatabase(t, adminURL, dbName),
		appURL:   withDatabase(t, testURL("MNEMO_TEST_DATABASE_URL", "mnemo_app"), dbName),
	}
}

func withDatabase(t *testing.T, raw, db string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	u.Path = "/" + db
	return u.String()
}

func (db testDB) migrate(t *testing.T) {
	t.Helper()
	if err := Migrate(t.Context(), db.adminURL); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func connect(t *testing.T, dbURL string) *pgx.Conn {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), dbURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func queryStrings(t *testing.T, conn *pgx.Conn, sql string, args ...any) []string {
	t.Helper()
	rows, err := conn.Query(t.Context(), sql, args...)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return got
}

func latestVersion(t *testing.T) int {
	t.Helper()
	ms, err := listMigrations(embedded)
	if err != nil {
		t.Fatalf("listMigrations: %v", err)
	}
	return ms[len(ms)-1].version
}

func TestMigrate_CreatesExtensions(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)

	got := queryStrings(t, connect(t, db.adminURL),
		"SELECT extname FROM pg_extension WHERE extname IN ('vector', 'pg_trgm') ORDER BY extname")
	if want := []string{"pg_trgm", "vector"}; !slices.Equal(got, want) {
		t.Errorf("extensions = %v, want %v", got, want)
	}
}

func TestMigrate_TablesMatchDataModel(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)
	conn := connect(t, db.adminURL)

	// Columns of docs/architecture.md §6, as "name type nullability".
	want := map[string][]string{
		"tenants": {"id uuid NO", "name text NO", "fts_config regconfig NO", "created_at timestamptz NO"},
		"subjects": {"tenant_id uuid NO", "id text NO", "policy jsonb NO", "purged_at timestamptz YES",
			"created_at timestamptz NO"},
		"episodes": {"id uuid NO", "tenant_id uuid NO", "subject_id text NO", "session_id text NO", "seq int4 NO",
			"role text NO", "content text NO", "occurred_at timestamptz NO", "recorded_at timestamptz NO",
			"trust text NO", "embedding vector YES", "fts tsvector YES", "token_count int4 YES", "api_key_id uuid YES"},
	}
	for table, cols := range want {
		got := queryStrings(t, conn, `SELECT column_name || ' ' || udt_name || ' ' || is_nullable
			FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position`, table)
		if !slices.Equal(got, cols) {
			t.Errorf("%s columns = %v, want %v", table, got, cols)
		}
	}
}

func TestMigrate_EmbeddingHas1024Dimensions(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)

	got := queryStrings(t, connect(t, db.adminURL), `SELECT format_type(atttypid, atttypmod)
		FROM pg_attribute WHERE attrelid = 'public.episodes'::regclass AND attname = 'embedding'`)
	if want := []string{"vector(1024)"}; !slices.Equal(got, want) {
		t.Errorf("episodes.embedding type = %v, want %v", got, want)
	}
}

func TestMigrate_InsertsDevTenant(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)

	got := queryStrings(t, connect(t, db.adminURL), "SELECT id::text FROM tenants")
	if want := []string{"00000000-0000-0000-0000-000000000001"}; !slices.Equal(got, want) {
		t.Errorf("tenants = %v, want %v", got, want)
	}
}

func TestMigrate_RecordsLatestVersion(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)

	got := queryStrings(t, connect(t, db.adminURL), "SELECT max(version)::text FROM schema_migrations")
	if want := []string{strconv.Itoa(latestVersion(t))}; !slices.Equal(got, want) {
		t.Errorf("schema_migrations max(version) = %v, want %v", got, want)
	}
}

func TestMigrate_SecondRunIsNoop(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)
	conn := connect(t, db.adminURL)
	const state = `SELECT format('%s %s', version, applied_at) FROM schema_migrations
		UNION ALL SELECT format('tenants %s', count(*)) FROM tenants`
	before := queryStrings(t, conn, state)

	db.migrate(t)

	if after := queryStrings(t, conn, state); !slices.Equal(after, before) {
		t.Errorf("state after second run = %v, want %v", after, before)
	}
}

func TestMigrate_FailedFileRollsBack(t *testing.T) {
	db := newTestDB(t)
	fsys := fstest.MapFS{
		"migrations/0001_ok.sql":  {Data: []byte("CREATE TABLE first_table (id int);")},
		"migrations/0002_bad.sql": {Data: []byte("CREATE TABLE second_table (id int); SELECT 1 / 0;")},
	}

	if err := migrate(t.Context(), db.adminURL, fsys); err == nil || !strings.Contains(err.Error(), "0002_bad.sql") {
		t.Fatalf("migrate error = %v, want one naming 0002_bad.sql", err)
	}

	conn := connect(t, db.adminURL)
	tables := queryStrings(t, conn, `SELECT table_name::text FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name IN ('first_table', 'second_table')`)
	if want := []string{"first_table"}; !slices.Equal(tables, want) {
		t.Errorf("tables = %v, want %v", tables, want)
	}
	versions := queryStrings(t, conn, "SELECT version::text FROM schema_migrations")
	if want := []string{"1"}; !slices.Equal(versions, want) {
		t.Errorf("schema_migrations = %v, want %v", versions, want)
	}
}

// TestMigrate_PgvectorVersionCheck covers the version rule and the query that
// reads the version from a real server; it does not call Migrate itself.
func TestMigrate_PgvectorVersionCheck(t *testing.T) {
	db := newTestDB(t)
	if err := requirePgvector(t.Context(), connect(t, db.adminURL)); err != nil {
		t.Errorf("requirePgvector on the development image = %v, want nil", err)
	}
	for _, version := range []string{"", "0.7.4", "0.5", "x.y"} {
		err := checkPgvector(version)
		if err == nil || !strings.Contains(err.Error(), "pgvector >= 0.8") {
			t.Errorf("checkPgvector(%q) = %v, want an error naming pgvector >= 0.8", version, err)
		}
	}
	for _, version := range []string{"0.8.0", "0.8.7", "0.10.1", "1.0.0"} {
		if err := checkPgvector(version); err != nil {
			t.Errorf("checkPgvector(%q) = %v, want nil", version, err)
		}
	}
}

func TestMigrate_InvalidURLHidesPassword(t *testing.T) {
	const password = "S3cr@t"
	err := Migrate(t.Context(), "postgres://u:"+password+"@127.0.0.1:1/db?sslmode=bogus")
	if err == nil {
		t.Fatal("Migrate with an invalid URL = nil, want an error")
	}
	if msg := err.Error(); strings.Contains(msg, "S3cr") || strings.Contains(msg, "t@127.0.0.1") {
		t.Errorf("Migrate error = %q, leaks part of the password", msg)
	}
}

func TestMigrate_OpenPoolInvalidURLHidesPassword(t *testing.T) {
	const password = "S3cr@t"
	_, err := OpenPool(t.Context(), "postgres://u:"+password+"@127.0.0.1:1/db?sslmode=bogus")
	if err == nil {
		t.Fatal("OpenPool with an invalid URL = nil, want an error")
	}
	if msg := err.Error(); strings.Contains(msg, "S3cr") || strings.Contains(msg, "t@127.0.0.1") {
		t.Errorf("OpenPool error = %q, leaks part of the password", msg)
	}
}

func TestMigrate_OpenPoolSetsStatementTimeout(t *testing.T) {
	db := newTestDB(t)

	var got string
	if err := appPool(t, db).QueryRow(t.Context(), "SHOW statement_timeout").Scan(&got); err != nil {
		t.Fatalf("SHOW statement_timeout: %v", err)
	}
	if got != "5s" {
		t.Errorf("statement_timeout = %q, want 5s", got)
	}
}

func appPool(t *testing.T, db testDB) *pgxpool.Pool {
	t.Helper()
	pool, err := OpenPool(t.Context(), db.appURL)
	if err != nil {
		t.Fatalf("OpenPool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestMigrate_ReadyAtLatestVersion(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)

	if err := Ready(t.Context(), appPool(t, db)); err != nil {
		t.Errorf("Ready as mnemo_app = %v, want nil", err)
	}
}

func TestMigrate_NotReadyWithoutMigrations(t *testing.T) {
	db := newTestDB(t)

	if err := Ready(t.Context(), appPool(t, db)); err == nil {
		t.Error("Ready on an unmigrated database = nil, want an error")
	}
}

func TestMigrate_NotReadyBehindLatest(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)
	if _, err := connect(t, db.adminURL).Exec(t.Context(),
		"DELETE FROM schema_migrations WHERE version = $1", latestVersion(t)); err != nil {
		t.Fatalf("delete latest version: %v", err)
	}

	if err := Ready(t.Context(), appPool(t, db)); err == nil {
		t.Error("Ready one version behind = nil, want an error")
	}
}

func TestMigrate_ReadyAheadOfBinary(t *testing.T) {
	db := newTestDB(t)
	db.migrate(t)
	if _, err := connect(t, db.adminURL).Exec(t.Context(),
		"INSERT INTO schema_migrations (version) VALUES ($1)", latestVersion(t)+1); err != nil {
		t.Fatalf("insert a newer version: %v", err)
	}

	if err := Ready(t.Context(), appPool(t, db)); err != nil {
		t.Errorf("Ready with a newer schema = %v, want nil", err)
	}
}
