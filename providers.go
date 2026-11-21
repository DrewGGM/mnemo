package mnemo

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"
)

// Usage is what one model call consumed.
type Usage struct {
	InputTokens  int
	OutputTokens int
	Provider     string
	Model        string
}

// Embedder turns texts into vectors of Dim dimensions, one per text and in
// the order of texts.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, Usage, error)
	Dim() int
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
