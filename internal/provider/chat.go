// Package provider talks to model providers over the OpenAI-compatible HTTP
// protocol with the standard library only (ADR-0006): every call has a
// timeout and bounded retries, and reports what it consumed.
package provider

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const chatTimeout = 30 * time.Second

// Usage is what one model call consumed.
type Usage struct {
	InputTokens  int
	OutputTokens int
	// Provider is the host name of the endpoint called.
	Provider string
	Model    string
}

// ChatRequest is one system and user turn. With JSONSchema set, the model
// must answer with JSON that conforms to it, in strict mode.
type ChatRequest struct {
	System     string
	User       string
	JSONSchema []byte
	MaxTokens  int
}

// ChatResponse is the text of the first choice and the call's usage.
type ChatResponse struct {
	Text  string
	Usage Usage
}

// Chat calls POST {baseURL}/chat/completions.
type Chat struct {
	ep endpoint
}

// NewChat checks baseURL and model; apiKey may be empty for a local server.
func NewChat(baseURL, apiKey, model string) (*Chat, error) {
	ep, err := newEndpoint(baseURL, "/chat/completions", apiKey, model, chatTimeout)
	if err != nil {
		return nil, err
	}
	return &Chat{ep: ep}, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type jsonSchemaFormat struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict"`
}

type responseFormat struct {
	Type       string           `json:"type"`
	JSONSchema jsonSchemaFormat `json:"json_schema"`
}

type chatBody struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatAnswer struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// checkChoice rejects a text that is not a whole answer: cut at MaxTokens,
// blocked by the content filter, or refused. The error gives the reason only,
// never the text.
func checkChoice(finishReason, refusal string) error {
	switch {
	case refusal != "":
		return errors.New("chat completion: the model refused to answer")
	case finishReason == "length":
		return errors.New("chat completion: answer cut short at the token limit")
	case finishReason == "content_filter":
		return errors.New("chat completion: answer blocked by the content filter")
	}
	return nil
}

// Complete sends req and returns the first choice.
func (c *Chat) Complete(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := chatBody{Model: c.ep.model, MaxTokens: req.MaxTokens}
	if req.System != "" {
		body.Messages = append(body.Messages, chatMessage{Role: "system", Content: req.System})
	}
	body.Messages = append(body.Messages, chatMessage{Role: "user", Content: req.User})
	if req.JSONSchema != nil {
		body.ResponseFormat = &responseFormat{
			Type:       "json_schema",
			JSONSchema: jsonSchemaFormat{Name: "response", Schema: req.JSONSchema, Strict: true},
		}
	}
	var ans chatAnswer
	if err := c.ep.post(ctx, "chat completion", body, &ans); err != nil {
		return ChatResponse{}, err
	}
	if len(ans.Choices) == 0 {
		return ChatResponse{}, errors.New("chat completion: answer has no choices")
	}
	if err := checkChoice(ans.Choices[0].FinishReason, ans.Choices[0].Message.Refusal); err != nil {
		return ChatResponse{}, err
	}
	return ChatResponse{
		Text: ans.Choices[0].Message.Content,
		Usage: Usage{
			InputTokens:  ans.Usage.PromptTokens,
			OutputTokens: ans.Usage.CompletionTokens,
			Provider:     c.ep.provider,
			Model:        c.ep.model,
		},
	}, nil
}
