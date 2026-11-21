package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DrewGGM/mnemo"
	"github.com/DrewGGM/mnemo/internal/httpapi"
)

// memoryWithoutDatabase never reaches PostgreSQL: pgxpool dials lazily and
// every request here fails before storage.
func memoryWithoutDatabase(t *testing.T) *mnemo.Memory {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), "postgres://nobody@127.0.0.1:1/none?connect_timeout=1")
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

func post(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
	httpapi.NewServer(httpapi.Deps{Memory: memoryWithoutDatabase(t)}).ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error.Code
}

func TestEpisodes_BodyOver1MiBReturns413(t *testing.T) {
	body := `{"subject":"s","messages":[{"role":"user","content":"` + strings.Repeat("a", 1<<20) + `"}]}`
	rec := post(t, "/v1/episodes", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	if code := errorCode(t, rec); code != "PAYLOAD_TOO_LARGE" {
		t.Errorf("error code = %q, want PAYLOAD_TOO_LARGE", code)
	}
}

func TestEpisodes_MalformedJSONReturns422(t *testing.T) {
	for _, path := range []string{"/v1/episodes", "/v1/recall"} {
		rec := post(t, path, `{"subject":`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("POST %s status = %d, want %d", path, rec.Code, http.StatusUnprocessableEntity)
			continue
		}
		if code := errorCode(t, rec); code != "VALIDATION_FAILED" {
			t.Errorf("POST %s error code = %q, want VALIDATION_FAILED", path, code)
		}
	}
}

func TestEpisodes_InvalidMessageReturns422(t *testing.T) {
	rec := post(t, "/v1/episodes",
		`{"subject":"s","session_id":"s-1","messages":[{"role":"summary","content":"x","occurred_at":"2026-03-10T15:04:00Z"}]}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if code := errorCode(t, rec); code != "VALIDATION_FAILED" {
		t.Errorf("error code = %q, want VALIDATION_FAILED", code)
	}
}

// The server here has no Logger, so this also covers that a nil Logger
// does not break the error envelope.
func TestEpisodes_StorageUnavailableReturns503(t *testing.T) {
	for path, body := range map[string]string{
		"/v1/episodes": `{"subject":"s","session_id":"s-1","messages":[{"role":"user","content":"x","occurred_at":"2026-03-10T15:04:00Z"}]}`,
		"/v1/recall":   `{"subject":"s","query":"x","max_tokens":100}`,
	} {
		rec := post(t, path, body)
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("POST %s status = %d, want %d", path, rec.Code, http.StatusServiceUnavailable)
			continue
		}
		if code := errorCode(t, rec); code != "UNAVAILABLE" {
			t.Errorf("POST %s error code = %q, want UNAVAILABLE", path, code)
		}
	}
}

func TestRecall_EmptyQueryReturns422(t *testing.T) {
	rec := post(t, "/v1/recall", `{"subject":"s","query":"","max_tokens":100}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if code := errorCode(t, rec); code != "VALIDATION_FAILED" {
		t.Errorf("error code = %q, want VALIDATION_FAILED", code)
	}
}
