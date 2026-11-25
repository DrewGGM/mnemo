package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	maxRetries  = 3
	baseBackoff = 250 * time.Millisecond
	// maxResponseBytes bounds what one answer may allocate; a 1,024-dimension
	// vector is about 13 KB of JSON, so 32 MB leaves room for large batches.
	maxResponseBytes = 32 << 20
)

// endpoint posts JSON to one operation of an OpenAI-compatible API. Its
// errors never carry the API key: they name the operation, the URL, which
// holds no credentials (checked by newEndpoint), and the HTTP status, never
// the body of the answer, which may echo the key or the request.
type endpoint struct {
	url      string
	apiKey   string
	model    string
	provider string
	client   *http.Client
	// timeout bounds the whole call, retries and waits included, so the
	// latency a caller sees has a ceiling.
	timeout time.Duration
	wait    func(ctx context.Context, d time.Duration) error
}

func newEndpoint(baseURL, path, apiKey, model string, timeout time.Duration) (endpoint, error) {
	u, err := checkBaseURL(baseURL)
	if err != nil {
		return endpoint{}, err
	}
	if model == "" {
		return endpoint{}, errors.New("model name is required")
	}
	return endpoint{
		url:      strings.TrimSuffix(baseURL, "/") + path,
		apiKey:   apiKey,
		model:    model,
		provider: u.Hostname(),
		client:   &http.Client{CheckRedirect: noRedirects},
		timeout:  timeout,
		wait:     sleep,
	}, nil
}

// noRedirects hands a 3xx back as the answer, a final error: following it
// could send the API key over plain http, or the request body to another host.
func noRedirects(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// checkBaseURL accepts only an http or https URL without user information,
// query or fragment, so the URL can appear in errors and logs. Its errors do
// not quote the URL, which may hold the credentials it rejects.
func checkBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return nil, errors.New("base URL is not a valid URL")
	case u.Scheme != "http" && u.Scheme != "https":
		return nil, errors.New("base URL must use http or https")
	case u.Host == "":
		return nil, errors.New("base URL has no host")
	case u.User != nil:
		return nil, errors.New("base URL must not carry credentials; pass the API key separately")
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "":
		return nil, errors.New("base URL must not have a query or a fragment")
	}
	return u, nil
}

// post sends body and decodes a 200 answer into out, retrying 429, 5xx and
// network errors up to maxRetries times with growing waits.
func (e endpoint) post(ctx context.Context, op string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("%s: encode request: %w", op, err)
	}
	callCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	for attempt := 0; ; attempt++ {
		retry, err := e.attempt(callCtx, payload, out)
		if err == nil {
			return nil
		}
		if retry && attempt < maxRetries {
			werr := e.wait(callCtx, backoff(attempt))
			if werr == nil {
				continue
			}
			err = fmt.Errorf("%w; last attempt: %w", werr, err)
		}
		// Only callCtx expiring, not the caller's context, is our timeout.
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("%s: no complete answer within %s: %w", op, e.timeout, err)
		}
		return fmt.Errorf("%s: %w", op, err)
	}
}

// attempt reports whether a failed request is worth retrying.
func (e endpoint) attempt(ctx context.Context, payload []byte, out any) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.url, bytes.NewReader(payload))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return ctx.Err() == nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body := io.LimitReader(resp.Body, maxResponseBytes)
	if resp.StatusCode != http.StatusOK {
		// Drained so the connection can be reused for the retry.
		_, _ = io.Copy(io.Discard, body)
		status := resp.StatusCode
		return status == http.StatusTooManyRequests || status >= 500, fmt.Errorf("%s answered HTTP %d", e.url, status)
	}
	if err := json.NewDecoder(body).Decode(out); err != nil {
		return false, fmt.Errorf("decode answer from %s: %w", e.url, err)
	}
	return false, nil
}

// backoff doubles the wait on each retry and adds up to half of it as
// jitter, so the waits grow strictly and concurrent callers spread out.
func backoff(retry int) time.Duration {
	d := baseBackoff << retry
	return d + rand.N(d/2)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
