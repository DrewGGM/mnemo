package httpapi

import (
	"net/http"
	"time"

	"github.com/DrewGGM/mnemo"
)

type recallRequest struct {
	Subject   string `json:"subject"`
	Query     string `json:"query"`
	MaxTokens int    `json:"max_tokens"`
}

type recallResponse struct {
	Episodes  []episodeJSON `json:"episodes"`
	TimingsMS timingsJSON   `json:"timings_ms"`
}

type episodeJSON struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Seq        int       `json:"seq"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	OccurredAt time.Time `json:"occurred_at"`
	Score      float64   `json:"score"`
}

type timingsJSON struct {
	Embed      int64 `json:"embed"`
	Candidates int64 `json:"candidates"`
	Total      int64 `json:"total"`
}

func (d Deps) postRecall(w http.ResponseWriter, r *http.Request) {
	var req recallRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	var opts []mnemo.RecallOption
	if req.MaxTokens != 0 {
		opts = append(opts, mnemo.MaxTokens(req.MaxTokens))
	}
	res, err := d.Memory.Scope(devTenant, req.Subject).Recall(r.Context(), req.Query, opts...)
	if err != nil {
		d.fail(w, r, err)
		return
	}
	eps := make([]episodeJSON, len(res.Episodes))
	for i, e := range res.Episodes {
		eps[i] = episodeJSON{
			ID:         formatUUID(e.ID),
			SessionID:  e.SessionID,
			Seq:        e.Seq,
			Role:       e.Role,
			Content:    e.Content,
			OccurredAt: e.OccurredAt,
			Score:      e.Score,
		}
	}
	d.respond(w, r, http.StatusOK, recallResponse{
		Episodes: eps,
		TimingsMS: timingsJSON{
			Embed:      res.Timings.Embed.Milliseconds(),
			Candidates: res.Timings.Candidates.Milliseconds(),
			Total:      res.Timings.Total.Milliseconds(),
		},
	})
}
