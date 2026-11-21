// Package httpapi translates HTTP requests into calls on the mnemo package.
package httpapi

import (
	"context"
	"net/http"
	"time"
)

// readyTimeout bounds the readiness check so a hung database turns into a
// 503 before the orchestrator's own probe timeout.
const readyTimeout = 2 * time.Second

// Deps holds what the handlers delegate to.
type Deps struct {
	// Ready reports whether the service can take traffic. Its error never
	// reaches the response body.
	Ready func(ctx context.Context) error
}

// NewServer returns the handler for every route of the service.
func NewServer(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz(deps.Ready))
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
