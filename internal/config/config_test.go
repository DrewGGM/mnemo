package config_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/DrewGGM/mnemo/internal/config"
)

func validEnv() map[string]string {
	return map[string]string{
		"MNEMO_DATABASE_URL":   "postgres://mnemo_app@127.0.0.1:55432/mnemo",
		"MNEMO_LLM_BASE_URL":   "http://llm.local/v1",
		"MNEMO_LLM_API_KEY":    "chat-key",
		"MNEMO_LLM_MODEL":      "chat-model",
		"MNEMO_EMBED_BASE_URL": "http://embed.local/v1",
		"MNEMO_EMBED_API_KEY":  "embed-key",
		"MNEMO_EMBED_MODEL":    "embed-model",
		"MNEMO_EMBED_DIM":      "1024",
		"MNEMO_RERANK_URL":     "http://rerank.local",
	}
}

func getenv(env map[string]string) func(string) string {
	return func(name string) string { return env[name] }
}

func TestLoad_ReadsEveryVariable(t *testing.T) {
	env := validEnv()
	env["MNEMO_LISTEN_ADDR"] = "127.0.0.1:9090"

	cfg, err := config.Load(getenv(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := config.Config{
		DatabaseURL:  env["MNEMO_DATABASE_URL"],
		ListenAddr:   "127.0.0.1:9090",
		ChatBaseURL:  env["MNEMO_LLM_BASE_URL"],
		ChatAPIKey:   env["MNEMO_LLM_API_KEY"],
		ChatModel:    env["MNEMO_LLM_MODEL"],
		EmbedBaseURL: env["MNEMO_EMBED_BASE_URL"],
		EmbedAPIKey:  env["MNEMO_EMBED_API_KEY"],
		EmbedModel:   env["MNEMO_EMBED_MODEL"],
		EmbedDim:     1024,
		RerankURL:    env["MNEMO_RERANK_URL"],
	}
	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoad_ListenAddrDefaultsToLoopback(t *testing.T) {
	cfg, err := config.Load(getenv(validEnv()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:8080" {
		t.Errorf("ListenAddr = %q, want 127.0.0.1:8080", cfg.ListenAddr)
	}
}

func TestLoad_MissingDatabaseURLNamesVariable(t *testing.T) {
	env := validEnv()
	delete(env, "MNEMO_DATABASE_URL")

	_, err := config.Load(getenv(env))
	if err == nil || !strings.Contains(err.Error(), "MNEMO_DATABASE_URL") {
		t.Fatalf("Load() error = %v, want one naming MNEMO_DATABASE_URL", err)
	}
}

func TestLoad_EmbedDimOtherThan1024NamesVariable(t *testing.T) {
	for _, dim := range []string{"", "768", "1536", "abc", " 1024", "1024.0"} {
		env := validEnv()
		env["MNEMO_EMBED_DIM"] = dim

		_, err := config.Load(getenv(env))
		if err == nil || !strings.Contains(err.Error(), "MNEMO_EMBED_DIM") {
			t.Errorf("MNEMO_EMBED_DIM=%q: Load() error = %v, want one naming MNEMO_EMBED_DIM", dim, err)
		}
	}
}

func TestLoad_ReportsEveryInvalidVariable(t *testing.T) {
	_, err := config.Load(getenv(map[string]string{}))
	if err == nil {
		t.Fatal("Load() error = nil, want one")
	}
	for _, name := range []string{"MNEMO_DATABASE_URL", "MNEMO_EMBED_DIM"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("Load() error = %q, want it to name %s", err, name)
		}
	}
}

// The service logs its configuration at startup, so logging a Config is the
// path through which a key or the database password could leak.
func TestConfig_LogOmitsSecrets(t *testing.T) {
	const sentinel = "sentinel-7f3a9c"
	env := validEnv()
	for name := range env {
		if strings.HasSuffix(name, "_API_KEY") {
			env[name] = sentinel + "-" + name
		}
	}
	env["MNEMO_DATABASE_URL"] = "postgres://mnemo_app:" + sentinel + "@127.0.0.1:55432/mnemo"

	cfg, err := config.Load(getenv(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	var out bytes.Buffer
	slog.New(slog.NewJSONHandler(&out, nil)).Info("starting", "config", cfg)

	if strings.Contains(out.String(), sentinel) {
		t.Errorf("log output contains a secret: %s", out.String())
	}
	if !strings.Contains(out.String(), `"embed_model":"embed-model"`) {
		t.Errorf("log output lacks the non-secret settings: %s", out.String())
	}
}
