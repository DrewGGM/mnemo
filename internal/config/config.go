// Package config loads the service configuration from the environment and
// validates all of it before the process starts serving.
package config

import (
	"errors"
	"fmt"
	"log/slog"
)

// embedDim is fixed by the schema's vector(1024) columns (ADR-0006); a model
// with another dimension cannot be used without a re-embedding migration.
const embedDim = 1024

const defaultListenAddr = "127.0.0.1:8080"

// Config is the validated service configuration.
type Config struct {
	DatabaseURL  string
	ListenAddr   string
	ChatBaseURL  string
	ChatAPIKey   string
	ChatModel    string
	EmbedBaseURL string
	EmbedAPIKey  string
	EmbedModel   string
	EmbedDim     int
	RerankURL    string
}

// Load reads the MNEMO_* variables through getenv and returns every invalid
// one in a single error, so an operator fixes them all in one restart.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DatabaseURL:  getenv("MNEMO_DATABASE_URL"),
		ListenAddr:   getenv("MNEMO_LISTEN_ADDR"),
		ChatBaseURL:  getenv("MNEMO_LLM_BASE_URL"),
		ChatAPIKey:   getenv("MNEMO_LLM_API_KEY"),
		ChatModel:    getenv("MNEMO_LLM_MODEL"),
		EmbedBaseURL: getenv("MNEMO_EMBED_BASE_URL"),
		EmbedAPIKey:  getenv("MNEMO_EMBED_API_KEY"),
		EmbedModel:   getenv("MNEMO_EMBED_MODEL"),
		RerankURL:    getenv("MNEMO_RERANK_URL"),
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("MNEMO_DATABASE_URL is required"))
	}
	// Compared as text so that "01024" or " 1024" are rejected, not silently accepted.
	if dim := getenv("MNEMO_EMBED_DIM"); dim != fmt.Sprint(embedDim) {
		errs = append(errs, fmt.Errorf("MNEMO_EMBED_DIM must be %d, got %q", embedDim, dim))
	}
	cfg.EmbedDim = embedDim
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// LogValue keeps API keys, the database URL (it carries the password) and the
// provider URLs (they may carry credentials) out of every log line.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("listen_addr", c.ListenAddr),
		slog.String("chat_model", c.ChatModel),
		slog.String("embed_model", c.EmbedModel),
		slog.Int("embed_dim", c.EmbedDim),
		slog.Bool("rerank", c.RerankURL != ""),
	)
}
