package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DrewGGM/mnemo/internal/httpapi"
)

func serve(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	httpapi.NewServer(httpapi.Deps{}).ServeHTTP(rec, req)
	return rec
}

func TestHealthz_ReturnsOK(t *testing.T) {
	if rec := serve(t, http.MethodGet, "/healthz"); rec.Code != http.StatusOK {
		t.Errorf("GET /healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHealthz_RejectsOtherMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if rec := serve(t, method, "/healthz"); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /healthz status = %d, want %d", method, rec.Code, http.StatusMethodNotAllowed)
		}
	}
}
