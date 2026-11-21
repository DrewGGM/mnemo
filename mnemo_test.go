package mnemo_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DrewGGM/mnemo"
)

// unreachablePool connects nowhere: pgxpool dials lazily, so it only fails
// if a test reaches the database, which validation must prevent.
func unreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func embed(t *testing.T, e mnemo.Embedder, texts ...string) [][]float32 {
	t.Helper()
	vecs, _, err := e.Embed(t.Context(), texts)
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	return vecs
}

func cosine(a, b []float32) float64 {
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot // FakeEmbedder returns unit vectors
}

func TestFakeEmbedder_Deterministic(t *testing.T) {
	e := mnemo.FakeEmbedder(1024)
	first := embed(t, e, "I moved to Medellin")
	second := embed(t, mnemo.FakeEmbedder(1024), "I moved to Medellin")
	if !slices.Equal(first[0], second[0]) {
		t.Error("two embeddings of the same text differ")
	}
}

func TestFakeEmbedder_UnitVectorsOfDim(t *testing.T) {
	for _, text := range []string{"I moved to Medellin", "?!", "a a a"} {
		v := embed(t, mnemo.FakeEmbedder(64), text)[0]
		if len(v) != 64 {
			t.Errorf("Embed(%q) has %d dimensions, want 64", text, len(v))
		}
		if norm := math.Sqrt(cosine(v, v)); math.Abs(norm-1) > 1e-6 {
			t.Errorf("Embed(%q) has norm %v, want 1", text, norm)
		}
	}
}

func TestFakeEmbedder_SharedWordsAreCloser(t *testing.T) {
	vecs := embed(t, mnemo.FakeEmbedder(1024),
		"where did I move", "I moved to Medellin, where I live now", "my dog is called Toby")
	if near, far := cosine(vecs[0], vecs[1]), cosine(vecs[0], vecs[2]); near <= far {
		t.Errorf("similarity with a shared word %v is not above one without %v", near, far)
	}
}

func TestOpen_RejectsWrongDimension(t *testing.T) {
	_, err := mnemo.Open(t.Context(), unreachablePool(t), mnemo.Config{Embedder: mnemo.FakeEmbedder(768)})
	if err == nil {
		t.Fatal("Open accepted an embedder of 768 dimensions")
	}
}

func TestOpen_RequiresEmbedder(t *testing.T) {
	if _, err := mnemo.Open(t.Context(), unreachablePool(t), mnemo.Config{}); err == nil {
		t.Fatal("Open accepted a Config without Embedder")
	}
}

func openUnreachable(t *testing.T) *mnemo.Memory {
	t.Helper()
	mem, err := mnemo.Open(t.Context(), unreachablePool(t), mnemo.Config{Embedder: mnemo.FakeEmbedder(1024)})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return mem
}

func TestRemember_RejectsInvalidInput(t *testing.T) {
	at := time.Date(2026, 3, 10, 15, 4, 0, 0, time.UTC)
	ok := mnemo.Message{Role: "user", Content: "hello", OccurredAt: at}
	with := func(content string) []mnemo.Message {
		return []mnemo.Message{{Role: "user", Content: content, OccurredAt: at}}
	}
	cases := map[string]struct {
		subject, session string
		msgs             []mnemo.Message
	}{
		"no subject":          {"", "", []mnemo.Message{ok}},
		"subject too long":    {strings.Repeat("s", 257), "", []mnemo.Message{ok}},
		"NUL in subject":      {"s\x00", "", []mnemo.Message{ok}},
		"session too long":    {"s", strings.Repeat("s", 257), []mnemo.Message{ok}},
		"NUL in session":      {"s", "s\x00", []mnemo.Message{ok}},
		"no messages":         {"s", "", nil},
		"summary role":        {"s", "", []mnemo.Message{{Role: "summary", Content: "x", OccurredAt: at}}},
		"unknown role":        {"s", "", []mnemo.Message{ok, {Role: "admin", Content: "x", OccurredAt: at}}},
		"empty content":       {"s", "", with("")},
		"content over 32 KiB": {"s", "", with(strings.Repeat("a", 32769))},
		"NUL in content":      {"s", "", with("a\x00b")},
		"invalid UTF-8":       {"s", "", with("a\xffb")},
		"one bad of many":     {"s", "", []mnemo.Message{ok, ok, {Role: "tool", OccurredAt: at}}},
		"201 messages":        {"s", "", slices.Repeat([]mnemo.Message{ok}, 201)},
	}
	mem := openUnreachable(t)
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := mem.Scope([16]byte{15: 1}, c.subject).Remember(t.Context(), c.msgs, mnemo.WithSession(c.session))
			if !errors.Is(err, mnemo.ErrInvalidInput) {
				t.Errorf("Remember error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

// 200 messages pass validation, so the call reaches the unreachable pool.
func TestRemember_Accepts200Messages(t *testing.T) {
	ok := mnemo.Message{Role: "user", Content: "hello", OccurredAt: time.Date(2026, 3, 10, 15, 4, 0, 0, time.UTC)}
	_, err := openUnreachable(t).Scope([16]byte{15: 1}, "s").Remember(t.Context(), slices.Repeat([]mnemo.Message{ok}, 200))
	if !errors.Is(err, mnemo.ErrUnavailable) {
		t.Errorf("Remember of 200 messages = %v, want ErrUnavailable from the pool", err)
	}
}

type zeroEmbedder struct{}

func (zeroEmbedder) Dim() int { return 1024 }

func (zeroEmbedder) Embed(_ context.Context, texts []string) ([][]float32, mnemo.Usage, error) {
	vecs := make([][]float32, len(texts))
	for i := range vecs {
		vecs[i] = make([]float32, 1024)
	}
	return vecs, mnemo.Usage{}, nil
}

// A zero vector has no cosine distance, so it must fail before storage.
func TestRecall_RejectsZeroVector(t *testing.T) {
	mem, err := mnemo.Open(t.Context(), unreachablePool(t), mnemo.Config{Embedder: zeroEmbedder{}})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_, err = mem.Scope([16]byte{15: 1}, "s").Recall(t.Context(), "where")
	if err == nil || !strings.Contains(err.Error(), "all zeros") {
		t.Errorf("Recall with a zero query vector = %v, want an embed error", err)
	}
}

func TestRecall_RejectsInvalidInput(t *testing.T) {
	cases := map[string]struct {
		subject, query string
		opts           []mnemo.RecallOption
	}{
		"no subject":        {"", "where", nil},
		"empty query":       {"s", " \t", nil},
		"zero max_tokens":   {"s", "where", []mnemo.RecallOption{mnemo.MaxTokens(0)}},
		"negative max_tok.": {"s", "where", []mnemo.RecallOption{mnemo.MaxTokens(-5)}},
	}
	mem := openUnreachable(t)
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := mem.Scope([16]byte{15: 1}, c.subject).Recall(t.Context(), c.query, c.opts...)
			if !errors.Is(err, mnemo.ErrInvalidInput) {
				t.Errorf("Recall error = %v, want ErrInvalidInput", err)
			}
		})
	}
}
