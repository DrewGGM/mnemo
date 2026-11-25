package mnemo

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"unicode"

	"github.com/DrewGGM/mnemo/internal/provider"
)

// Usage is what one model call consumed.
type Usage struct {
	InputTokens  int
	OutputTokens int
	// Provider is the host name of the endpoint an OpenAI-compatible model
	// called, such as "generativelanguage.googleapis.com", or "fake".
	Provider string
	Model    string
}

// Embedder turns texts into vectors of Dim dimensions, one per text and in
// the order of texts.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, Usage, error)
	Dim() int
}

// ChatRequest is one system and user turn to a ChatModel. With JSONSchema
// set, the model must answer with JSON that conforms to it, in strict mode.
// MaxTokens caps the answer; zero leaves the provider's default.
type ChatRequest struct {
	System     string
	User       string
	JSONSchema []byte
	MaxTokens  int
}

// ChatResponse is the text a ChatModel answered and what it consumed.
type ChatResponse struct {
	Text  string
	Usage Usage
}

// ChatModel completes one chat turn.
type ChatModel interface {
	Complete(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// OpenAICompatibleChat returns a ChatModel that calls POST
// {baseURL}/chat/completions, with up to 3 retries on 429, 5xx and network
// errors, all within a 30 s limit per call. baseURL must be http or https
// without credentials, query or fragment; apiKey, sent as a bearer token,
// may be empty for a local server, and never appears in an error. Redirects
// are not followed. An answer cut at MaxTokens, filtered or refused is an
// error. If baseURL or model is invalid, every call returns that error.
func OpenAICompatibleChat(baseURL, apiKey, model string) ChatModel {
	c, err := provider.NewChat(baseURL, apiKey, model)
	if err != nil {
		return invalidModel{err: err}
	}
	return openAIChat{c: c}
}

// OpenAICompatibleEmbedder returns an Embedder that calls POST
// {baseURL}/embeddings, with up to 3 retries on 429, 5xx and network errors,
// all within a 5 s limit per call, and rejects vectors that are not dim long.
// baseURL and apiKey follow the rules of OpenAICompatibleChat. If baseURL,
// model or dim is invalid, every call returns that error.
func OpenAICompatibleEmbedder(baseURL, apiKey, model string, dim int) Embedder {
	e, err := provider.NewEmbedder(baseURL, apiKey, model, dim)
	if err != nil {
		return invalidModel{err: err, dim: dim}
	}
	return openAIEmbedder{e: e}
}

type openAIChat struct {
	c *provider.Chat
}

func (o openAIChat) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	resp, err := o.c.Complete(ctx, provider.ChatRequest(req))
	return ChatResponse{Text: resp.Text, Usage: Usage(resp.Usage)}, err
}

type openAIEmbedder struct {
	e *provider.Embedder
}

func (o openAIEmbedder) Dim() int { return o.e.Dim() }

func (o openAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, Usage, error) {
	vecs, usage, err := o.e.Embed(ctx, texts)
	return vecs, Usage(usage), err
}

// invalidModel is a ChatModel and an Embedder built from an invalid
// configuration: it fails every call with the reason.
type invalidModel struct {
	err error
	dim int
}

func (m invalidModel) Complete(context.Context, ChatRequest) (ChatResponse, error) {
	return ChatResponse{}, fmt.Errorf("mnemo: invalid chat model: %w", m.err)
}

func (m invalidModel) Dim() int { return m.dim }

func (m invalidModel) Embed(context.Context, []string) ([][]float32, Usage, error) {
	return nil, Usage{}, fmt.Errorf("mnemo: invalid embedder: %w", m.err)
}

// FakeEmbedder returns a deterministic Embedder that needs no network: each
// word adds to one hashed dimension, so texts that share words get close
// vectors. It is meant for tests. dim must be positive.
func FakeEmbedder(dim int) Embedder {
	if dim < 1 {
		panic("mnemo: FakeEmbedder dimension must be positive")
	}
	return fakeEmbedder{dim: dim}
}

type fakeEmbedder struct {
	dim int
}

func (f fakeEmbedder) Dim() int { return f.dim }

func (f fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, Usage, error) {
	vecs := make([][]float32, len(texts))
	tokens := 0
	for i, text := range texts {
		vecs[i] = f.vector(text)
		tokens += estimateTokens(text)
	}
	return vecs, Usage{InputTokens: tokens, Provider: "fake", Model: "fake"}, nil
}

func (f fakeEmbedder) vector(text string) []float32 {
	v := make([]float32, f.dim)
	dim := uint64(f.dim)
	for _, word := range strings.FieldsFunc(strings.ToLower(text), isWordSeparator) {
		h := hash64(word)
		// The top bit picks the sign, so colliding words do not always add up.
		v[h%dim] += float32(1 - 2*int(h>>63))
	}
	// A zero vector has no cosine distance (pgvector returns NaN), which
	// happens for a text without words or when colliding words cancel out.
	if !normalize(v) {
		v[hash64(text)%dim] = 1
	}
	return v
}

func isWordSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsNumber(r)
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s)) // a hash.Hash never returns an error
	return h.Sum64()
}

// normalize scales v to unit length and reports false if v is all zeros.
func normalize(v []float32) bool {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return false
	}
	norm := math.Sqrt(sum)
	for i := range v {
		v[i] = float32(float64(v[i]) / norm)
	}
	return true
}
