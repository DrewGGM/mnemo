//go:build integration

package mcpserver_test

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/DrewGGM/mnemo"
	"github.com/DrewGGM/mnemo/internal/mcpserver"
)

// devTenant is the tenant that 0001_init.sql inserts; each test isolates
// itself with a subject of its own.
var devTenant = [16]byte{15: 1}

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

// migrateOnce runs `mnemo migrate`, because this package may not import
// internal/store and the root package has no Migrate until T-030. The
// migration is idempotent and serialised by an advisory lock, so it is safe
// while other packages migrate the same database.
var migrateOnce = sync.OnceValues(func() ([]byte, error) {
	cmd := exec.Command("go", "run", "github.com/DrewGGM/mnemo/cmd/mnemo", "migrate")
	cmd.Env = append(os.Environ(), "MNEMO_ADMIN_DATABASE_URL="+testURL("MNEMO_TEST_ADMIN_DATABASE_URL", "mnemo_admin"))
	return cmd.CombinedOutput()
})

func newScope(t *testing.T) *mnemo.Scope {
	t.Helper()
	if out, err := migrateOnce(); err != nil {
		t.Fatalf("mnemo migrate: %v\n%s", err, out)
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
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return mem.Scope(devTenant, "test-"+hex.EncodeToString(b))
}

// connect joins an SDK client to srv over in-memory transports.
func connect(t *testing.T, srv *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverEnd, clientEnd := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(t.Context(), serverEnd, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "mnemo-test", Version: "v0"}, nil)
	cs, err := client.Connect(t.Context(), clientEnd, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func recallTool(t *testing.T, cs *mcp.ClientSession) *mcp.Tool {
	t.Helper()
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	i := slices.IndexFunc(res.Tools, func(tool *mcp.Tool) bool { return tool.Name == "recall" })
	if i < 0 {
		t.Fatalf("tools/list has no recall tool")
	}
	return res.Tools[i]
}

func callRecall(t *testing.T, cs *mcp.ClientSession, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: "recall", Arguments: args})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("result has %d content blocks, want 1", len(res.Content))
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want *mcp.TextContent", res.Content[0])
	}
	return text.Text
}

func TestRecall_ToolArguments(t *testing.T) {
	tool := recallTool(t, connect(t, mcpserver.New(newScope(t))))
	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	var s struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}
	for _, want := range []string{"query", "as_of", "max_tokens"} {
		if _, ok := s.Properties[want]; !ok {
			t.Errorf("input schema has no %q property: %s", want, schema)
		}
	}
	if !slices.Equal(s.Required, []string{"query"}) {
		t.Errorf("required = %v, want only query", s.Required)
	}
	if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
		t.Errorf("annotations = %+v, want read-only", tool.Annotations)
	}
}

func TestRecall_InputSchemaHasNoSubjectOrTenant(t *testing.T) {
	tool := recallTool(t, connect(t, mcpserver.New(newScope(t))))
	schema, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	// Searched in the whole schema, so a nested property cannot carry them either.
	for _, key := range []string{"subject", "tenant"} {
		if strings.Contains(string(schema), `"`+key+`"`) {
			t.Errorf("input schema contains the key %q: %s", key, schema)
		}
	}
}

func TestRecall_ReturnsWhatRememberStored(t *testing.T) {
	scope := newScope(t)
	occurred := time.Date(2026, 3, 10, 15, 4, 0, 0, time.UTC)
	msgs := []mnemo.Message{
		{Role: "user", Content: "I moved to Medellin last spring", OccurredAt: occurred},
		{Role: "user", Content: "my dog is called Toby", OccurredAt: occurred},
	}
	if _, err := scope.Remember(t.Context(), msgs, mnemo.WithSession("s-1")); err != nil {
		t.Fatalf("Remember: %v", err)
	}

	res := callRecall(t, connect(t, mcpserver.New(scope)), map[string]any{"query": "moved to Medellin"})
	text := resultText(t, res)
	if res.IsError {
		t.Fatalf("recall failed: %s", text)
	}
	var out struct {
		Episodes []struct {
			SessionID  string    `json:"session_id"`
			Seq        int       `json:"seq"`
			Role       string    `json:"role"`
			Content    string    `json:"content"`
			OccurredAt time.Time `json:"occurred_at"`
		} `json:"episodes"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("result is not the episodes JSON: %v\n%s", err, text)
	}
	if len(out.Episodes) != 2 {
		t.Fatalf("recall returned %d episodes, want 2: %s", len(out.Episodes), text)
	}
	first := out.Episodes[0]
	if first.Content != "I moved to Medellin last spring" || first.Role != "user" || first.SessionID != "s-1" || first.Seq != 1 {
		t.Errorf("first episode = %+v, want the remembered Medellin message, seq 1 of s-1", first)
	}
	if !first.OccurredAt.Equal(occurred) {
		t.Errorf("occurred_at = %v, want %v", first.OccurredAt, occurred)
	}
}

func TestRecall_AsOfIsRejected(t *testing.T) {
	cs := connect(t, mcpserver.New(newScope(t)))
	res := callRecall(t, cs, map[string]any{"query": "Medellin", "as_of": "2026-05-01T00:00:00Z"})
	if text := resultText(t, res); !res.IsError || !strings.Contains(text, "as_of") {
		t.Errorf("recall with as_of: IsError = %v, text %q; want a tool error that names as_of", res.IsError, text)
	}
}

func TestRecall_EmptyQueryIsToolError(t *testing.T) {
	cs := connect(t, mcpserver.New(newScope(t)))
	res := callRecall(t, cs, map[string]any{"query": " "})
	if text := resultText(t, res); !res.IsError || !strings.Contains(text, "query") {
		t.Errorf("recall with a blank query: IsError = %v, text %q; want a tool error that names query", res.IsError, text)
	}
}

func TestRecall_MaxTokensOverCapIsToolError(t *testing.T) {
	cs := connect(t, mcpserver.New(newScope(t)))
	res := callRecall(t, cs, map[string]any{"query": "Medellin", "max_tokens": 4001})
	if text := resultText(t, res); !res.IsError || !strings.Contains(text, "max_tokens") {
		t.Errorf("recall with max_tokens 4001: IsError = %v, text %q; want a tool error that names max_tokens", res.IsError, text)
	}
}
