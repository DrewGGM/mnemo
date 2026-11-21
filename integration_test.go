//go:build integration

package mnemo_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DrewGGM/mnemo"
	"github.com/DrewGGM/mnemo/internal/store"
)

// devTenant is the tenant that 0001_init.sql inserts; each test isolates
// itself with subjects of its own.
var devTenant = [16]byte{15: 1}

var occurred = time.Date(2026, 3, 10, 15, 4, 0, 0, time.UTC)

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

// migrateOnce brings the shared database to the latest schema, so the tests
// pass whether or not the admin service of Compose ran first. Migrate is
// idempotent and serialised by an advisory lock.
var migrateOnce = sync.OnceValue(func() error {
	return store.Migrate(context.Background(), testURL("MNEMO_TEST_ADMIN_DATABASE_URL", "mnemo_admin"))
})

func openMemory(t *testing.T) *mnemo.Memory {
	t.Helper()
	if err := migrateOnce(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	pool, err := pgxpool.New(t.Context(), testURL("MNEMO_TEST_DATABASE_URL", "mnemo_app"))
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	mem, err := mnemo.Open(t.Context(), pool, mnemo.Config{Embedder: mnemo.FakeEmbedder(1024)})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return mem
}

func newSubject(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "test-" + hex.EncodeToString(b)
}

func messages(contents ...string) []mnemo.Message {
	msgs := make([]mnemo.Message, len(contents))
	for i, c := range contents {
		msgs[i] = mnemo.Message{Role: "user", Content: c, OccurredAt: occurred}
	}
	return msgs
}

func remember(t *testing.T, s *mnemo.Scope, session string, contents ...string) [][16]byte {
	t.Helper()
	res, err := s.Remember(t.Context(), messages(contents...), mnemo.WithSession(session))
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	if len(res.EpisodeIDs) != len(contents) {
		t.Fatalf("Remember returned %d ids for %d messages", len(res.EpisodeIDs), len(contents))
	}
	return res.EpisodeIDs
}

func recall(t *testing.T, s *mnemo.Scope, query string, opts ...mnemo.RecallOption) []mnemo.Episode {
	t.Helper()
	res, err := s.Recall(t.Context(), query, opts...)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	return res.Episodes
}

func TestEpisodes_SeqConsecutiveWithinSession(t *testing.T) {
	s := openMemory(t).Scope(devTenant, newSubject(t))
	first := remember(t, s, "s-1", "I moved to Medellin", "I work as a nurse")
	second := remember(t, s, "s-1", "I have a dog")
	other := remember(t, s, "s-2", "I like coffee")

	seq := map[[16]byte]int{}
	for _, e := range recall(t, s, "I", mnemo.MaxTokens(1000)) {
		seq[e.ID] = e.Seq
	}
	want := map[[16]byte]int{first[0]: 1, first[1]: 2, second[0]: 3, other[0]: 1}
	for id, w := range want {
		if got, ok := seq[id]; !ok || got != w {
			t.Errorf("episode %x: seq = %d (found %v), want %d", id, got, ok, w)
		}
	}
}

func TestEpisodes_MissingOccurredAtUsesReceptionTime(t *testing.T) {
	s := openMemory(t).Scope(devTenant, newSubject(t))
	if _, err := s.Remember(t.Context(), []mnemo.Message{{Role: "user", Content: "I moved to Medellin"}}); err != nil {
		t.Fatalf("Remember without occurred_at: %v", err)
	}
	got := recall(t, s, "Medellin")
	if len(got) != 1 {
		t.Fatalf("recall returned %d episodes, want 1", len(got))
	}
	if at := got[0].OccurredAt; at.IsZero() || at.Year() < 2026 || at.Location() != time.UTC {
		t.Errorf("occurred_at = %v, want the reception time in UTC", at)
	}
}

func TestEpisodes_OtherSubjectSeesNone(t *testing.T) {
	mem := openMemory(t)
	owner := mem.Scope(devTenant, newSubject(t))
	remember(t, owner, "s-1", "I moved to Medellin", "my dog is called Toby")
	if got := recall(t, owner, "Medellin"); len(got) != 2 {
		t.Fatalf("the owner recalls %d episodes, want 2", len(got))
	}
	if got := recall(t, mem.Scope(devTenant, newSubject(t)), "Medellin"); len(got) != 0 {
		t.Errorf("another subject of the tenant recalls %d episodes, want 0", len(got))
	}
}

func TestRecall_Deterministic(t *testing.T) {
	s := openMemory(t).Scope(devTenant, newSubject(t))
	// Repeated contents tie on distance, so the order also depends on how ties break.
	remember(t, s, "s-1", "I moved to Medellin", "I moved to Medellin", "Medellin is warm",
		"my dog is called Toby", "I moved to Medellin")
	ids := func() [][16]byte {
		var out [][16]byte
		for _, e := range recall(t, s, "moved to Medellin") {
			out = append(out, e.ID)
		}
		return out
	}
	first, second := ids(), ids()
	if len(first) != 5 {
		t.Fatalf("recall returned %d episodes, want 5", len(first))
	}
	if !slices.Equal(first, second) {
		t.Errorf("two recalls returned different orders:\n%x\n%x", first, second)
	}
}

func TestRecall_OrdersByCosineSimilarity(t *testing.T) {
	s := openMemory(t).Scope(devTenant, newSubject(t))
	remember(t, s, "s-1", "my dog is called Toby", "I moved to Medellin last spring", "coffee every morning")
	got := recall(t, s, "moved to Medellin")
	if len(got) != 3 {
		t.Fatalf("recall returned %d episodes, want 3", len(got))
	}
	if got[0].Content != "I moved to Medellin last spring" {
		t.Errorf("first episode = %q, want the one that shares the query's words", got[0].Content)
	}
	if at := got[0].OccurredAt; at.Location() != time.UTC || !at.Equal(occurred) {
		t.Errorf("occurred_at = %v, want %v in UTC", at, occurred)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Score > got[i-1].Score {
			t.Errorf("score %v at %d is above %v at %d", got[i].Score, i, got[i-1].Score, i-1)
		}
	}
}

func TestRecall_StopsAtMaxTokens(t *testing.T) {
	s := openMemory(t).Scope(devTenant, newSubject(t))
	// 40 characters each: 10 estimated tokens.
	remember(t, s, "s-1",
		"Medellin aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"Medellin bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"Medellin ccccccccccccccccccccccccccccccc")
	if got := recall(t, s, "Medellin", mnemo.MaxTokens(25)); len(got) != 2 {
		t.Errorf("recall with 25 tokens returned %d episodes of 10 tokens, want 2", len(got))
	}
}
