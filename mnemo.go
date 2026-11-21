// Package mnemo is long-term memory for LLM agents, stored in PostgreSQL:
// it keeps every message of a subject as an episode and recalls the ones
// closest to a query.
package mnemo

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DrewGGM/mnemo/internal/store"
)

// embedDim is fixed by the vector(1024) columns of the schema (ADR-0006).
const embedDim = 1024

// recallCandidates is the most episodes the vector search returns
// (docs/architecture.md §9).
const recallCandidates = 40

const defaultMaxTokens = 1500

// maxContentBytes matches the CHECK on episodes.content in 0001_init.sql.
const maxContentBytes = 32768

// maxIDBytes bounds subject and session identifiers, which are keys of
// btree indexes with a size limit per entry.
const maxIDBytes = 256

// maxMessages is the most messages one Remember call takes (TH-11).
const maxMessages = 200

// ErrInvalidInput reports arguments that Remember or Recall reject before
// touching storage; the error text names the offending field.
var ErrInvalidInput = errors.New("invalid input")

// ErrUnavailable reports that the database could not be reached or did not
// answer in time; retrying later may succeed.
var ErrUnavailable = store.ErrUnavailable

// Config holds the models Memory uses.
type Config struct {
	// Embedder embeds episodes and queries. Required; its Dim must be 1024.
	Embedder Embedder
}

// Memory stores and recalls the memory of every tenant in one database.
type Memory struct {
	pool     *pgxpool.Pool
	embedder Embedder
}

// Open returns a Memory over pool, a pool of the mnemo_app role on a
// migrated database.
func Open(_ context.Context, pool *pgxpool.Pool, cfg Config) (*Memory, error) {
	if pool == nil {
		return nil, errors.New("mnemo: Open needs a pool")
	}
	if cfg.Embedder == nil {
		return nil, errors.New("mnemo: Config.Embedder is required")
	}
	if dim := cfg.Embedder.Dim(); dim != embedDim {
		return nil, fmt.Errorf("mnemo: embedder dimension is %d, the schema needs %d", dim, embedDim)
	}
	return &Memory{pool: pool, embedder: cfg.Embedder}, nil
}

// Scope is the memory of one subject of one tenant.
type Scope struct {
	m     *Memory
	scope store.Scope
}

// Scope returns the memory of subjectID in tenantID.
func (m *Memory) Scope(tenantID [16]byte, subjectID string) *Scope {
	return &Scope{m: m, scope: store.Scope{TenantID: tenantID, SubjectID: subjectID}}
}

// Remember stores each message as an immutable episode with its embedding,
// numbered consecutively within its session.
func (s *Scope) Remember(ctx context.Context, msgs []Message, opts ...RememberOption) (RememberResult, error) {
	var o rememberOptions
	for _, opt := range opts {
		opt(&o)
	}
	if err := s.validateRemember(msgs, o.sessionID); err != nil {
		return RememberResult{}, err
	}
	texts := make([]string, len(msgs))
	for i, msg := range msgs {
		texts[i] = msg.Content
	}
	vecs, err := s.m.embed(ctx, texts)
	if err != nil {
		return RememberResult{}, err
	}
	received := time.Now().UTC()
	eps := make([]store.NewEpisode, len(msgs))
	for i, msg := range msgs {
		eps[i] = store.NewEpisode{
			Role:       msg.Role,
			Content:    msg.Content,
			Trust:      trustOf(msg.Role),
			OccurredAt: msg.OccurredAt,
			Embedding:  vecs[i],
		}
		if msg.OccurredAt.IsZero() {
			eps[i].OccurredAt = received
		}
	}
	ids, err := store.InsertEpisodes(ctx, s.m.pool, s.scope, o.sessionID, eps)
	if err != nil {
		return RememberResult{}, err
	}
	return RememberResult{EpisodeIDs: ids}, nil
}

func (s *Scope) validateRemember(msgs []Message, sessionID string) error {
	if err := validateID("subject", s.scope.SubjectID); err != nil {
		return err
	}
	if !validText(sessionID) || len(sessionID) > maxIDBytes {
		return fmt.Errorf("%w: session_id must be UTF-8 text of at most %d bytes without NUL bytes", ErrInvalidInput, maxIDBytes)
	}
	switch {
	case len(msgs) == 0:
		return fmt.Errorf("%w: no messages", ErrInvalidInput)
	case len(msgs) > maxMessages:
		return fmt.Errorf("%w: more than %d messages", ErrInvalidInput, maxMessages)
	}
	for i, msg := range msgs {
		switch {
		case !validRole(msg.Role):
			return fmt.Errorf("%w: messages[%d]: role must be user, assistant, tool or system", ErrInvalidInput, i)
		case msg.Content == "":
			return fmt.Errorf("%w: messages[%d]: content is empty", ErrInvalidInput, i)
		case len(msg.Content) > maxContentBytes:
			return fmt.Errorf("%w: messages[%d]: content exceeds %d bytes", ErrInvalidInput, i, maxContentBytes)
		case !validText(msg.Content):
			return fmt.Errorf("%w: messages[%d]: content is not UTF-8 text without NUL bytes", ErrInvalidInput, i)
		}
	}
	return nil
}

// validText reports whether PostgreSQL accepts s as text: valid UTF-8
// without NUL bytes.
func validText(s string) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}

// validateID rejects identifiers that PostgreSQL would refuse as text or
// that are too long for an entry of its btree indexes.
func validateID(field, id string) error {
	switch {
	case id == "":
		return fmt.Errorf("%w: %s is empty", ErrInvalidInput, field)
	case len(id) > maxIDBytes:
		return fmt.Errorf("%w: %s exceeds %d bytes", ErrInvalidInput, field, maxIDBytes)
	case !validText(id):
		return fmt.Errorf("%w: %s is not UTF-8 text without NUL bytes", ErrInvalidInput, field)
	}
	return nil
}

// validRole leaves out summary: only consolidation writes summaries.
func validRole(role string) bool {
	switch role {
	case "user", "assistant", "tool", "system":
		return true
	}
	return false
}

// trustOf derives an episode's trust from its role: tool output and system
// text may carry instructions planted by a third party (TH-3).
func trustOf(role string) string {
	switch role {
	case "user", "assistant":
		return role
	}
	return "untrusted"
}

// Recall returns the subject's episodes ordered by cosine similarity to
// query, most similar first, up to the token budget.
func (s *Scope) Recall(ctx context.Context, query string, opts ...RecallOption) (RecallResult, error) {
	o := recallOptions{maxTokens: defaultMaxTokens}
	for _, opt := range opts {
		opt(&o)
	}
	if err := validateID("subject", s.scope.SubjectID); err != nil {
		return RecallResult{}, err
	}
	switch {
	case strings.TrimSpace(query) == "":
		return RecallResult{}, fmt.Errorf("%w: query is empty", ErrInvalidInput)
	case o.maxTokens < 1:
		return RecallResult{}, fmt.Errorf("%w: max_tokens must be positive", ErrInvalidInput)
	}

	start := time.Now()
	vecs, err := s.m.embed(ctx, []string{query})
	if err != nil {
		return RecallResult{}, err
	}
	embedded := time.Now()
	found, err := store.SearchEpisodesByVector(ctx, s.m.pool, s.scope, vecs[0], recallCandidates)
	if err != nil {
		return RecallResult{}, err
	}
	searched := time.Now()
	return RecallResult{
		Episodes: withinBudget(found, o.maxTokens),
		Timings: Timings{
			Embed:      embedded.Sub(start),
			Candidates: searched.Sub(embedded),
			Total:      time.Since(start),
		},
	}, nil
}

// withinBudget keeps episodes in order while their estimated tokens fit in
// maxTokens.
func withinBudget(found []store.FoundEpisode, maxTokens int) []Episode {
	eps := make([]Episode, 0, len(found))
	used := 0
	for _, f := range found {
		used += estimateTokens(f.Content)
		if used > maxTokens {
			break
		}
		eps = append(eps, Episode(f))
	}
	return eps
}

func (m *Memory) embed(ctx context.Context, texts []string) ([][]float32, error) {
	vecs, _, err := m.embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	if len(vecs) != len(texts) {
		return nil, fmt.Errorf("embed: got %d vectors for %d texts", len(vecs), len(texts))
	}
	// pgvector gives NaN as the cosine distance of a zero vector, and a NaN
	// score cannot be written as JSON.
	for i, v := range vecs {
		if len(v) != embedDim || !slices.ContainsFunc(v, func(x float32) bool { return x != 0 }) {
			return nil, fmt.Errorf("embed: vector %d has %d dimensions or is all zeros, want %d nonzero", i, len(v), embedDim)
		}
	}
	return vecs, nil
}

// estimateTokens counts characters / 4, rounded up (AD-4).
func estimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + 3) / 4
}
