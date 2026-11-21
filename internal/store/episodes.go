package store

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewEpisode is one message to store, already embedded.
type NewEpisode struct {
	Role       string
	Content    string
	Trust      string
	OccurredAt time.Time
	Embedding  []float32
}

// FoundEpisode is an episode returned by a search, with its cosine
// similarity to the query.
type FoundEpisode struct {
	ID         [16]byte
	SessionID  string
	Seq        int
	Role       string
	Content    string
	OccurredAt time.Time
	Score      float64
}

// The tsvector uses the tenant's text search configuration, so a tenant set
// to spanish gets Spanish stemming.
const insertEpisodesSQL = `
INSERT INTO episodes (tenant_id, subject_id, session_id, seq, role, content, occurred_at, trust, embedding, fts)
SELECT $1, $2, $3, $4 + m.ord, m.role, m.content, m.occurred_at, m.trust, m.embedding::vector,
       to_tsvector((SELECT fts_config FROM tenants WHERE id = $1), m.content)
FROM unnest($5::text[], $6::text[], $7::timestamptz[], $8::text[], $9::text[])
     WITH ORDINALITY AS m(role, content, occurred_at, trust, embedding, ord)
RETURNING id, seq`

const searchEpisodesSQL = `
SELECT id, session_id, seq, role, content, occurred_at, 1 - (embedding <=> $3::vector) AS score
FROM episodes
WHERE tenant_id = $1 AND subject_id = $2 AND embedding IS NOT NULL
ORDER BY embedding <=> $3::vector, id
LIMIT $4`

// InsertEpisodes stores eps in session sessionID, numbered after the last
// episode of that session, and returns their identifiers in the order of eps.
// It creates the subject on its first write.
func InsertEpisodes(ctx context.Context, pool *pgxpool.Pool, s Scope, sessionID string, eps []NewEpisode) ([][16]byte, error) {
	var ids [][16]byte
	err := WithScope(ctx, pool, s, func(tx pgx.Tx) error {
		last, err := lockSubject(ctx, tx, s, sessionID)
		if err != nil {
			return err
		}
		ids, err = insertEpisodes(ctx, tx, s, sessionID, last, eps)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("insert episodes: %w", err)
	}
	return ids, nil
}

// lockSubject takes the subject's row lock, so concurrent writes of the
// same subject number their episodes one after the other, and returns the
// last sequence number of the session (0 if it has none).
func lockSubject(ctx context.Context, tx pgx.Tx, s Scope, sessionID string) (int, error) {
	if _, err := tx.Exec(ctx,
		"INSERT INTO subjects (tenant_id, id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		s.TenantID, s.SubjectID); err != nil {
		return 0, fmt.Errorf("create subject: %w", err)
	}
	if _, err := tx.Exec(ctx,
		"SELECT 1 FROM subjects WHERE tenant_id = $1 AND id = $2 FOR UPDATE",
		s.TenantID, s.SubjectID); err != nil {
		return 0, fmt.Errorf("lock subject: %w", err)
	}
	var last int
	err := tx.QueryRow(ctx,
		"SELECT coalesce(max(seq), 0) FROM episodes WHERE tenant_id = $1 AND subject_id = $2 AND session_id = $3",
		s.TenantID, s.SubjectID, sessionID).Scan(&last)
	if err != nil {
		return 0, fmt.Errorf("read last seq: %w", err)
	}
	return last, nil
}

func insertEpisodes(ctx context.Context, tx pgx.Tx, s Scope, sessionID string, last int, eps []NewEpisode) ([][16]byte, error) {
	n := len(eps)
	roles, contents, trusts, vectors := make([]string, n), make([]string, n), make([]string, n), make([]string, n)
	occurred := make([]time.Time, n)
	for i, e := range eps {
		roles[i], contents[i], trusts[i] = e.Role, e.Content, e.Trust
		occurred[i] = e.OccurredAt
		vectors[i] = vectorLiteral(e.Embedding)
	}
	rows, err := tx.Query(ctx, insertEpisodesSQL,
		s.TenantID, s.SubjectID, sessionID, last, roles, contents, occurred, trusts, vectors)
	if err != nil {
		return nil, err
	}
	type inserted struct {
		ID  [16]byte
		Seq int
	}
	got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[inserted])
	if err != nil {
		return nil, err
	}
	// RETURNING does not promise the order of the SELECT; seq does.
	slices.SortFunc(got, func(a, b inserted) int { return cmp.Compare(a.Seq, b.Seq) })
	ids := make([][16]byte, len(got))
	for i, g := range got {
		ids[i] = g.ID
	}
	return ids, nil
}

// SearchEpisodesByVector returns up to limit episodes of the subject ordered
// by cosine distance to query, nearest first, with ties broken by identifier
// so the same data always gives the same order.
func SearchEpisodesByVector(ctx context.Context, pool *pgxpool.Pool, s Scope, query []float32, limit int) ([]FoundEpisode, error) {
	var found []FoundEpisode
	err := WithScope(ctx, pool, s, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, searchEpisodesSQL, s.TenantID, s.SubjectID, vectorLiteral(query), limit)
		if err != nil {
			return err
		}
		found, err = pgx.CollectRows(rows, pgx.RowToStructByPos[FoundEpisode])
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("search episodes by vector: %w", err)
	}
	// pgx returns timestamptz values in time.Local.
	for i := range found {
		found[i].OccurredAt = found[i].OccurredAt.UTC()
	}
	return found, nil
}

// vectorLiteral writes v in pgvector's text format, which PostgreSQL casts
// to vector, so pgvector-go is not needed (security-baseline §8).
func vectorLiteral(v []float32) string {
	var b strings.Builder
	b.Grow(len(v) * 12)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'g', -1, 32))
	}
	b.WriteByte(']')
	return b.String()
}
