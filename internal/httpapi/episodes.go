package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/DrewGGM/mnemo"
)

// maxBodyBytes caps every request body (TH-11).
const maxBodyBytes = 1 << 20

// devTenant is the development tenant inserted by 0001_init.sql. Every /v1
// request uses it until requests carry a credential (T-012).
var devTenant = [16]byte{15: 1}

type episodesRequest struct {
	Subject   string        `json:"subject"`
	SessionID string        `json:"session_id"`
	Messages  []messageJSON `json:"messages"`
}

type messageJSON struct {
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	OccurredAt time.Time `json:"occurred_at"`
}

type episodesResponse struct {
	EpisodeIDs []string `json:"episode_ids"`
}

func (d Deps) postEpisodes(w http.ResponseWriter, r *http.Request) {
	var req episodesRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	msgs := make([]mnemo.Message, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = mnemo.Message{Role: m.Role, Content: m.Content, OccurredAt: m.OccurredAt}
	}
	res, err := d.Memory.Scope(devTenant, req.Subject).Remember(r.Context(), msgs, mnemo.WithSession(req.SessionID))
	if err != nil {
		d.fail(w, r, err)
		return
	}
	ids := make([]string, len(res.EpisodeIDs))
	for i, id := range res.EpisodeIDs {
		ids[i] = formatUUID(id)
	}
	d.respond(w, r, http.StatusAccepted, episodesResponse{EpisodeIDs: ids})
}

// decodeJSON reads the body into dst and, if it cannot, writes the error
// response and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(dst)
	if err == nil {
		return true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body exceeds 1 MiB")
		return false
	}
	writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "request body is not valid JSON for this route")
	return false
}

// fail answers an error of the mnemo package. Only validation errors reach
// the client; the rest are logged, never echoed, because they may carry SQL.
func (d Deps) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, mnemo.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", err.Error())
		return
	}
	d.Logger.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	if errors.Is(err, mnemo.ErrUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "storage unavailable")
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	// Strings and an empty map always encode.
	b, _ := json.Marshal(errorBody{Error: errorDetail{Code: code, Message: message, Details: map[string]any{}}})
	writeBody(w, status, b)
}

// respond encodes body before sending the status, so a value that JSON
// cannot hold, such as a NaN score, becomes a 500 and not a 200 with an
// empty body.
func (d Deps) respond(w http.ResponseWriter, r *http.Request, status int, body any) {
	b, err := json.Marshal(body)
	if err != nil {
		d.fail(w, r, fmt.Errorf("encode response: %w", err))
		return
	}
	writeBody(w, status, b)
}

func writeBody(w http.ResponseWriter, status int, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(b, '\n')) // the status is already sent
}

func formatUUID(id [16]byte) string {
	var b [36]byte
	hex.Encode(b[0:8], id[0:4])
	b[8] = '-'
	hex.Encode(b[9:13], id[4:6])
	b[13] = '-'
	hex.Encode(b[14:18], id[6:8])
	b[18] = '-'
	hex.Encode(b[19:23], id[8:10])
	b[23] = '-'
	hex.Encode(b[24:36], id[10:16])
	return string(b[:])
}
