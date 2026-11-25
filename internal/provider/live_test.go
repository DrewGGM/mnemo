//go:build live

package provider

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The live tests call the real providers named by the MNEMO_LLM_* and
// MNEMO_EMBED_* variables; they cost money and never run in CI:
//
//	go test -tags=live ./internal/provider -run TestLive_ -count=1 -v
func liveEnv(t *testing.T, prefix string) (baseURL, apiKey, model string) {
	t.Helper()
	baseURL, apiKey, model = os.Getenv(prefix+"_BASE_URL"), os.Getenv(prefix+"_API_KEY"), os.Getenv(prefix+"_MODEL")
	// The key may be empty: a local server may need none.
	if baseURL == "" || model == "" {
		t.Fatalf("%s_BASE_URL and %s_MODEL are required by the live tests", prefix, prefix)
	}
	return baseURL, apiKey, model
}

func TestLive_ChatAnswersWithSchemaJSON(t *testing.T) {
	c, err := NewChat(liveEnv(t, "MNEMO_LLM"))
	if err != nil {
		t.Fatal(err)
	}
	schema := `{"type":"object","properties":{"capital":{"type":"string"}},"required":["capital"],"additionalProperties":false}`
	resp, err := c.Complete(t.Context(), ChatRequest{
		System:     "Answer with JSON only.",
		User:       "What is the capital of France?",
		JSONSchema: []byte(schema),
		MaxTokens:  64,
	})
	if err != nil {
		t.Fatal(err)
	}
	var out struct{ Capital string }
	dec := json.NewDecoder(strings.NewReader(resp.Text))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil || out.Capital == "" {
		t.Fatalf("answer %q does not match the schema: %v", resp.Text, err)
	}
	t.Logf("capital=%q, usage %+v", out.Capital, resp.Usage)
}

func TestLive_EmbedderReturns1024Dimensions(t *testing.T) {
	baseURL, apiKey, model := liveEnv(t, "MNEMO_EMBED")
	e, err := NewEmbedder(baseURL, apiKey, model, 1024)
	if err != nil {
		t.Fatal(err)
	}
	vecs, usage, err := e.Embed(t.Context(), []string{"I moved to Bogota last spring.", "Where do I live?"})
	if err != nil {
		t.Fatal(err)
	}
	// With dim 1024, Embed fails on a vector of any other length, so a nil
	// error is the dimension check.
	t.Logf("%d vectors, usage %+v", len(vecs), usage)
}
