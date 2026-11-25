package provider

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const embedTimeout = 5 * time.Second

// Embedder calls POST {baseURL}/embeddings.
type Embedder struct {
	ep  endpoint
	dim int
}

// NewEmbedder checks baseURL, model and dim; apiKey may be empty for a local
// server.
func NewEmbedder(baseURL, apiKey, model string, dim int) (*Embedder, error) {
	if dim < 1 {
		return nil, errors.New("embedding dimension must be positive")
	}
	ep, err := newEndpoint(baseURL, "/embeddings", apiKey, model, embedTimeout)
	if err != nil {
		return nil, err
	}
	return &Embedder{ep: ep, dim: dim}, nil
}

// Dim is the length of every vector Embed returns.
func (e *Embedder) Dim() int { return e.dim }

type embedBody struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format"`
}

type embedAnswer struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
	} `json:"usage"`
}

// Embed returns one vector per text, in the order of texts, and rejects an
// answer whose vectors are missing, repeated or not Dim long.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, Usage, error) {
	var ans embedAnswer
	body := embedBody{Model: e.ep.model, Input: texts, EncodingFormat: "float"}
	if err := e.ep.post(ctx, "embeddings", body, &ans); err != nil {
		return nil, Usage{}, err
	}
	if len(ans.Data) != len(texts) {
		return nil, Usage{}, fmt.Errorf("embeddings: got %d vectors for %d texts", len(ans.Data), len(texts))
	}
	vecs := make([][]float32, len(texts))
	for _, d := range ans.Data {
		if d.Index < 0 || d.Index >= len(texts) || vecs[d.Index] != nil {
			return nil, Usage{}, fmt.Errorf("embeddings: vector index %d is out of range or repeated", d.Index)
		}
		if len(d.Embedding) != e.dim {
			return nil, Usage{}, fmt.Errorf("embeddings: vector %d has %d dimensions, want %d", d.Index, len(d.Embedding), e.dim)
		}
		vecs[d.Index] = d.Embedding
	}
	return vecs, Usage{InputTokens: ans.Usage.PromptTokens, Provider: e.ep.provider, Model: e.ep.model}, nil
}
