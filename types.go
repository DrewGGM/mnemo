package mnemo

import "time"

// Message is one turn of a conversation to remember.
type Message struct {
	// Role is user, assistant, tool or system.
	Role    string
	Content string
	// OccurredAt is when the message was said, not when it is stored. If it
	// is zero, Remember uses the time it receives the message.
	OccurredAt time.Time
}

// RememberResult identifies the episodes that Remember stored, in the order
// of its messages.
type RememberResult struct {
	EpisodeIDs [][16]byte
}

// Episode is a stored message returned by Recall.
type Episode struct {
	ID         [16]byte
	SessionID  string
	Seq        int
	Role       string
	Content    string
	OccurredAt time.Time
	// Score is the cosine similarity between the episode and the query.
	Score float64
}

// RecallResult holds the episodes of the subject closest to the query, most
// similar first, and how long each stage took.
type RecallResult struct {
	Episodes []Episode
	Timings  Timings
}

// Timings is the duration of each stage of Recall.
type Timings struct {
	Embed      time.Duration
	Candidates time.Duration
	Total      time.Duration
}

// RememberOption changes how Remember stores messages.
type RememberOption func(*rememberOptions)

type rememberOptions struct {
	sessionID string
}

// WithSession stores the messages in session id. Sequence numbers are
// consecutive within a session; without this option the session is "".
func WithSession(id string) RememberOption {
	return func(o *rememberOptions) { o.sessionID = id }
}

// RecallOption changes what Recall returns.
type RecallOption func(*recallOptions)

type recallOptions struct {
	maxTokens int
}

// MaxTokens caps the estimated tokens of the returned episodes; n must be
// positive. The default is 1500.
func MaxTokens(n int) RecallOption {
	return func(o *recallOptions) { o.maxTokens = n }
}
