//go:build e2e

// Package e2e drives the service started with `docker compose up -d --wait`.
package e2e

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// baseURL is the mnemo service of docker-compose.yml, or MNEMO_E2E_BASE_URL.
func baseURL() string {
	if u := os.Getenv("MNEMO_E2E_BASE_URL"); u != "" {
		return u
	}
	return "http://127.0.0.1:8080"
}

func postJSON(t *testing.T, path string, body, out any) int {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, baseURL()+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s (is `docker compose up -d --wait` running?): %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("POST %s: decode response: %v", path, err)
	}
	return resp.StatusCode
}

type recalled struct {
	Episodes []struct {
		ID      string  `json:"id"`
		Seq     int     `json:"seq"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"episodes"`
	TimingsMS map[string]int64 `json:"timings_ms"`
}

func TestSkeleton_RememberThenRecall(t *testing.T) {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	subject := "e2e-" + hex.EncodeToString(b)
	contents := []string{"my dog is called Toby", "I moved to Medellin last spring", "coffee every morning"}
	msgs := make([]map[string]string, len(contents))
	for i, c := range contents {
		msgs[i] = map[string]string{"role": "user", "content": c, "occurred_at": "2026-03-10T15:04:00Z"}
	}

	var written struct {
		EpisodeIDs []string `json:"episode_ids"`
	}
	status := postJSON(t, "/v1/episodes", map[string]any{"subject": subject, "session_id": "s-1", "messages": msgs}, &written)
	if status != http.StatusAccepted || len(written.EpisodeIDs) != len(contents) {
		t.Fatalf("POST /v1/episodes = %d with %d ids, want 202 with %d", status, len(written.EpisodeIDs), len(contents))
	}

	var got recalled
	status = postJSON(t, "/v1/recall", map[string]any{"subject": subject, "query": "moved to Medellin", "max_tokens": 500}, &got)
	if status != http.StatusOK || len(got.Episodes) != len(contents) {
		t.Fatalf("POST /v1/recall = %d with %d episodes, want 200 with %d", status, len(got.Episodes), len(contents))
	}
	if top := got.Episodes[0]; top.ID != written.EpisodeIDs[1] || top.Seq != 2 || top.Content != contents[1] {
		t.Errorf("first recalled episode = %+v, want id %s, seq 2, %q", top, written.EpisodeIDs[1], contents[1])
	}
	if _, ok := got.TimingsMS["total"]; !ok {
		t.Errorf("timings_ms = %v, want a total", got.TimingsMS)
	}
}
