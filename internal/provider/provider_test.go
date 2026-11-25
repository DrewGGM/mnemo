package provider

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testKey   = "test-api-key-must-never-leak"
	testModel = "test-model"
	testDim   = 3
)

const chatOK = `{"choices":[{"message":{"content":"{\"answer\":\"Paris\"}"}}],"usage":{"prompt_tokens":12,"completion_tokens":5}}`

const embedOK = `{"data":[{"index":0,"embedding":[0.1,0.2,0.3]},{"index":1,"embedding":[0.4,0.5,0.6]}],"usage":{"prompt_tokens":4,"total_tokens":4}}`

// target is one adapter driven through the same HTTP failure cases.
type target struct {
	name string
	path string
	ok   string
	call func(t *testing.T, baseURL string, tune func(*endpoint)) error
}

var targets = []target{
	{name: "chat", path: "/chat/completions", ok: chatOK, call: callChat},
	{name: "embeddings", path: "/embeddings", ok: embedOK, call: callEmbed},
}

func callChat(t *testing.T, baseURL string, tune func(*endpoint)) error {
	c, err := NewChat(baseURL, testKey, testModel)
	if err != nil {
		t.Fatalf("NewChat: %v", err)
	}
	tune(&c.ep)
	_, err = c.Complete(t.Context(), ChatRequest{User: "hello"})
	return err
}

func callEmbed(t *testing.T, baseURL string, tune func(*endpoint)) error {
	e, err := NewEmbedder(baseURL, testKey, testModel, testDim)
	if err != nil {
		t.Fatalf("NewEmbedder: %v", err)
	}
	tune(&e.ep)
	_, _, err = e.Embed(t.Context(), []string{"hello", "world"})
	return err
}

// fakeAPI answers each request with the next of answers, repeating the last
// one, counts the requests and keeps the body and Authorization of the last.
type fakeAPI struct {
	srv        *httptest.Server
	requests   atomic.Int32
	body, auth string
}

type answer struct {
	status int
	body   string
}

func newFakeAPI(t *testing.T, path string, answers ...answer) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := int(f.requests.Add(1))
		if r.Method != http.MethodPost || r.URL.Path != path {
			t.Errorf("got %s %s, want POST %s", r.Method, r.URL.Path, path)
		}
		body, _ := io.ReadAll(r.Body)
		f.body, f.auth = string(body), strings.Join(r.Header.Values("Authorization"), ",")
		a := answers[min(n, len(answers))-1]
		w.Header().Set("Location", path)
		w.WriteHeader(a.status)
		_, _ = io.WriteString(w, a.body)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// recordWaits replaces the real sleep, so retries take no time and the
// waits they ask for can be inspected.
func recordWaits(waits *[]time.Duration) func(*endpoint) {
	return func(ep *endpoint) {
		ep.wait = func(_ context.Context, d time.Duration) error {
			*waits = append(*waits, d)
			return nil
		}
	}
}

func TestChat_DecodesAnswer(t *testing.T) {
	api := newFakeAPI(t, "/chat/completions", answer{http.StatusOK, chatOK})
	c, err := NewChat(api.srv.URL+"/", testKey, testModel)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Complete(t.Context(), ChatRequest{User: "Capital of France?"})
	if err != nil {
		t.Fatal(err)
	}
	want := ChatResponse{
		Text:  `{"answer":"Paris"}`,
		Usage: Usage{InputTokens: 12, OutputTokens: 5, Provider: "127.0.0.1", Model: testModel},
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestChat_SendsRequest(t *testing.T) {
	schema := `{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`
	cases := []struct {
		name, key, wantAuth string
		req                 ChatRequest
		want                string
	}{
		{
			name: "full", key: testKey, wantAuth: "Bearer " + testKey,
			req: ChatRequest{System: "Be brief.", User: "Capital of France?", JSONSchema: []byte(schema), MaxTokens: 64},
			want: `{"model":"test-model","max_tokens":64,
				"messages":[{"role":"system","content":"Be brief."},{"role":"user","content":"Capital of France?"}],
				"response_format":{"type":"json_schema","json_schema":{"name":"response","strict":true,"schema":` + schema + `}}}`,
		},
		{
			name: "user only without key",
			req:  ChatRequest{User: "Hi"},
			want: `{"model":"test-model","messages":[{"role":"user","content":"Hi"}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := newFakeAPI(t, "/chat/completions", answer{http.StatusOK, chatOK})
			c, err := NewChat(api.srv.URL, tc.key, testModel)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Complete(t.Context(), tc.req); err != nil {
				t.Fatal(err)
			}
			assertSameJSON(t, api.body, tc.want)
			if api.auth != tc.wantAuth {
				t.Errorf("Authorization header is %q, want %q", api.auth, tc.wantAuth)
			}
		})
	}
}

func TestChat_RejectsIncompleteAnswer(t *testing.T) {
	cases := map[string]string{
		"no choices":     `[]`,
		"token limit":    `[{"message":{"content":"{\"ans"},"finish_reason":"length"}]`,
		"content filter": `[{"message":{"content":""},"finish_reason":"content_filter"}]`,
		"refusal":        `[{"message":{"content":"","refusal":"I can't help with that."},"finish_reason":"stop"}]`,
	}
	for name, choices := range cases {
		t.Run(name, func(t *testing.T) {
			api := newFakeAPI(t, "/chat/completions", answer{http.StatusOK, `{"choices":` + choices + `}`})
			if err := callChat(t, api.srv.URL, func(*endpoint) {}); err == nil {
				t.Fatal("got no error")
			}
		})
	}
}

func TestEmbedder_DecodesAnswer(t *testing.T) {
	// Vectors come back out of order; index says which text each belongs to.
	const body = `{"data":[{"index":1,"embedding":[4,5,6]},{"index":0,"embedding":[1,2,3]}],
		"usage":{"prompt_tokens":9,"total_tokens":9}}`
	api := newFakeAPI(t, "/embeddings", answer{http.StatusOK, body})
	e, err := NewEmbedder(api.srv.URL, testKey, testModel, testDim)
	if err != nil {
		t.Fatal(err)
	}
	vecs, usage, err := e.Embed(t.Context(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if want := [][]float32{{1, 2, 3}, {4, 5, 6}}; !reflect.DeepEqual(vecs, want) {
		t.Errorf("vectors are %v, want %v", vecs, want)
	}
	if want := (Usage{InputTokens: 9, Provider: "127.0.0.1", Model: testModel}); usage != want {
		t.Errorf("usage is %+v, want %+v", usage, want)
	}
	assertSameJSON(t, api.body, `{"model":"test-model","input":["first","second"],"encoding_format":"float"}`)
}

func TestEmbedder_RejectsMalformedAnswer(t *testing.T) {
	const ok = `{"index":0,"embedding":[1,2,3]}`
	cases := map[string]string{
		"short vector":   ok + `,{"index":1,"embedding":[1,2]}`,
		"long vector":    ok + `,{"index":1,"embedding":[1,2,3,4]}`,
		"null vector":    ok + `,{"index":1,"embedding":null}`,
		"missing vector": ok,
		"index too high": ok + `,{"index":2,"embedding":[1,2,3]}`,
		"repeated index": ok + "," + ok,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			api := newFakeAPI(t, "/embeddings", answer{http.StatusOK, `{"data":[` + data + `]}`})
			if err := callEmbed(t, api.srv.URL, func(*endpoint) {}); err == nil {
				t.Fatal("got no error")
			}
		})
	}
}

func TestPost_RetriesRetryableStatus(t *testing.T) {
	for _, tg := range targets {
		for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
			t.Run(tg.name+"/"+http.StatusText(status), func(t *testing.T) {
				api := newFakeAPI(t, tg.path, answer{status, "busy"})
				var waits []time.Duration
				err := tg.call(t, api.srv.URL, recordWaits(&waits))
				if err == nil {
					t.Fatal("got no error after every attempt failed")
				}
				if n := api.requests.Load(); n != 1+maxRetries {
					t.Errorf("made %d requests, want %d", n, 1+maxRetries)
				}
				assertGrowing(t, waits, maxRetries)
			})
		}
	}
}

func TestPost_SucceedsAfterRetry(t *testing.T) {
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			api := newFakeAPI(t, tg.path, answer{http.StatusServiceUnavailable, ""}, answer{http.StatusOK, tg.ok})
			if err := tg.call(t, api.srv.URL, recordWaits(new([]time.Duration))); err != nil {
				t.Fatal(err)
			}
			if n := api.requests.Load(); n != 2 {
				t.Errorf("made %d requests, want 2", n)
			}
		})
	}
}

func TestPost_RetriesNetworkError(t *testing.T) {
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			srv := httptest.NewServer(http.NotFoundHandler())
			srv.Close() // nothing listens on its address any more
			var waits []time.Duration
			if err := tg.call(t, srv.URL, recordWaits(&waits)); err == nil {
				t.Fatal("got no error from a closed server")
			}
			assertGrowing(t, waits, maxRetries)
		})
	}
}

func TestPost_DoesNotRetryFinalErrors(t *testing.T) {
	finals := map[string]answer{
		"bad request":  {http.StatusBadRequest, `{"error":{"message":"bad"}}`},
		"invalid JSON": {http.StatusOK, `{"choices":`},
		// Followed, it would loop back to the server and be counted.
		"redirect": {http.StatusTemporaryRedirect, ""},
	}
	for _, tg := range targets {
		for name, a := range finals {
			t.Run(tg.name+"/"+name, func(t *testing.T) {
				api := newFakeAPI(t, tg.path, a)
				var waits []time.Duration
				if err := tg.call(t, api.srv.URL, recordWaits(&waits)); err == nil {
					t.Fatal("got no error")
				}
				if n := api.requests.Load(); n != 1 || len(waits) != 0 {
					t.Errorf("made %d requests and %d waits, want 1 and 0", n, len(waits))
				}
			})
		}
	}
}

func TestPost_TimesOut(t *testing.T) {
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			// Once the body is read, the server notices the client hanging up and
			// cancels the request context, so the handler returns and Close does not block.
			srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				<-r.Context().Done()
			}))
			defer srv.Close()
			err := tg.call(t, srv.URL, func(ep *endpoint) { ep.timeout = 50 * time.Millisecond })
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("got error %v, want a deadline exceeded", err)
			}
		})
	}
}

func TestPost_TimeoutDuringWaitKeepsLastError(t *testing.T) {
	api := newFakeAPI(t, "/embeddings", answer{http.StatusServiceUnavailable, ""})
	err := callEmbed(t, api.srv.URL, func(ep *endpoint) {
		ep.wait = func(context.Context, time.Duration) error { return context.DeadlineExceeded }
	})
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "HTTP 503") {
		t.Fatalf("got error %v, want a deadline exceeded that names HTTP 503", err)
	}
}

func TestErrors_NeverContainAPIKey(t *testing.T) {
	// The server echoes the credential, as some providers do in their errors:
	// a final 400 for chat, and 500s retried until they run out for embeddings.
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusBadRequest
		if r.URL.Path == "/embeddings" {
			status = http.StatusInternalServerError
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"error":{"message":"invalid key `+r.Header.Get("Authorization")+`"}}`)
	}))
	defer echo.Close()
	var errs []error
	for _, tg := range targets {
		errs = append(errs, tg.call(t, echo.URL, recordWaits(new([]time.Duration))))
	}
	for _, base := range []string{"https://user:" + testKey + "@example.com/v1", "https://example.com/v1?key=" + testKey} {
		_, chatErr := NewChat(base, testKey, testModel)
		_, embedErr := NewEmbedder(base, testKey, testModel, testDim)
		errs = append(errs, chatErr, embedErr)
	}
	for i, err := range errs {
		if err == nil || strings.Contains(err.Error(), testKey) {
			t.Errorf("case %d: got error %v, want one without the API key", i, err)
		}
	}
}

func TestCheckBaseURL(t *testing.T) {
	valid := []string{"http://127.0.0.1:8000/v1", "https://api.example.com/v1/", "https://example.com/v1beta/openai"}
	invalid := []string{
		"", "example.com/v1", "ftp://example.com/v1", "file:///etc/passwd", "https:///v1",
		"https://user@example.com/v1", "https://user:pass@example.com/v1",
		"https://example.com/v1?key=k", "https://example.com/v1?", "https://example.com/v1#f", "http://[::1",
	}
	for i, raw := range append(valid, invalid...) {
		if _, err := checkBaseURL(raw); (err == nil) != (i < len(valid)) {
			t.Errorf("%q: want valid %t, got error %v", raw, i < len(valid), err)
		}
	}
}

func TestNew_ChecksConfig(t *testing.T) {
	const base = "https://example.com/v1"
	_, chatErr := NewChat(base, testKey, "")
	_, embedErr := NewEmbedder(base, testKey, "", testDim)
	_, dimErr := NewEmbedder(base, testKey, testModel, 0)
	if chatErr == nil || embedErr == nil || dimErr == nil {
		t.Errorf("no model or dimension 0 accepted: errors are %v, %v and %v", chatErr, embedErr, dimErr)
	}
	c, chatErr := NewChat(base, "", testModel)
	e, embedErr := NewEmbedder(base, "", testModel, testDim)
	if chatErr != nil || embedErr != nil {
		t.Fatal(chatErr, embedErr)
	}
	if c.ep.timeout != 30*time.Second || e.ep.timeout != 5*time.Second {
		t.Errorf("timeouts are %s for chat and %s for embeddings, want 30s and 5s", c.ep.timeout, e.ep.timeout)
	}
}

func assertGrowing(t *testing.T, waits []time.Duration, n int) {
	t.Helper()
	if len(waits) != n {
		t.Fatalf("waited %d times, want %d", len(waits), n)
	}
	for i := 1; i < len(waits); i++ {
		if waits[i] <= waits[i-1] {
			t.Errorf("wait %d is %s, not longer than the previous %s", i, waits[i], waits[i-1])
		}
	}
}

func assertSameJSON(t *testing.T, got, want string) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("expected body is not JSON: %v", err)
	}
	if json.Unmarshal([]byte(got), &g) != nil || !reflect.DeepEqual(g, w) {
		t.Errorf("request body is %s, want %s", got, want)
	}
}
