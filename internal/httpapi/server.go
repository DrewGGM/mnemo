// Package httpapi translates HTTP requests into calls on the mnemo package.
package httpapi

import "net/http"

// Deps holds what the handlers delegate to.
type Deps struct{}

// NewServer returns the handler for every route of the service.
func NewServer(Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	return mux
}

// healthz reports liveness only: it never touches the database, so an
// orchestrator does not restart the process when PostgreSQL is down.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}
