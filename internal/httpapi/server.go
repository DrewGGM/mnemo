// Package httpapi translates HTTP requests into calls on the mnemo package.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/DrewGGM/mnemo"
)

// readyTimeout bounds the readiness check so a hung database turns into a
// 503 before the orchestrator's own probe timeout.
const readyTimeout = 2 * time.Second

// Deps holds what the handlers delegate to.
type Deps struct {
	// Ready reports whether the service can take traffic. Its error never
	// reaches the response body.
	Ready func(ctx context.Context) error
	// Memory serves the /v1 routes.
	Memory *mnemo.Memory
	// Logger records the errors that a response hides. Nil discards them.
	Logger *slog.Logger
}

// NewServer returns the handler for every route of the service.
func NewServer(deps Deps) http.Handler {
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.DiscardHandler)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(deps.Ready))
	mux.HandleFunc("POST /v1/episodes", deps.postEpisodes)
	mux.HandleFunc("POST /v1/recall", deps.postRecall)
	return mux
}

// healthz reports liveness only: it never touches the database, so an
// orchestrator does not restart the process when PostgreSQL is down.
func healthz(w http.ResponseWriter, _ *http.Request) {
	writeText(w, http.StatusOK, "ok\n")
}

func readyz(ready func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		if err := ready(ctx); err != nil {
			writeText(w, http.StatusServiceUnavailable, "not ready\n")
			return
		}
		writeText(w, http.StatusOK, "ok\n")
	}
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
