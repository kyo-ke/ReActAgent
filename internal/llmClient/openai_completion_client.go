package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/kyo/AIAgent/internal/logging"
)

// OpenAICompletionClient implements Chat Completions API:
// POST /v1/chat/completions
// Reference: https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create
//
// Env:
// - OPENAI_API_KEY (required)
// - OPENAI_MODEL (default: gpt-4.1-mini)
// - OPENAI_BASE_URL (default: https://api.openai.com)
//
// This client intentionally targets *only* the Completions API (not Responses API).
// It supports function tools.
type OpenAICompletionClient struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
	Log        logging.Logger
}

func NewOpenAICompletionClientFromEnv() *OpenAICompletionClient {
	return NewOpenAICompletionClientFromEnvWithLogger(nil)
}

// NewOpenAICompletionClientFromEnvWithLogger is the DI-friendly constructor.
// If log is nil, a default stderr logger is created.
func NewOpenAICompletionClientFromEnvWithLogger(log logging.Logger) *OpenAICompletionClient {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4.1-mini"
	}
	base := os.Getenv("OPENAI_BASE_URL")
	if base == "" {
		base = "https://api.openai.com"
	}
	if log == nil {
		log = logging.New()
	}
	return &OpenAICompletionClient{
		HTTPClient: http.DefaultClient,
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		BaseURL:    base,
		Model:      model,
		Log:        log,
	}
}

func (c *OpenAICompletionClient) ensureDefaults(call LLMRequest) (LLMRequest, error) {
	if c.APIKey == "" {
		return LLMRequest{}, fmt.Errorf("openai: missing OPENAI_API_KEY")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Log == nil {
		c.Log = logging.New()
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com"
	}
	if c.Model == "" {
		c.Model = "gpt-4.1-mini"
	}
	if call.Model == "" {
		call.Model = c.Model
	}
	return call, nil
}

func (c *OpenAICompletionClient) CallLLM(ctx context.Context, call LLMRequest) (LLMResult, error) {
	call, err := c.ensureDefaults(call)
	if err != nil {
		return LLMResult{}, err
	}

	c.Log.Debugf("openai chat.completions: model=%s messages=%d tools=%d", call.Model, len(call.Messages), len(call.Tools))

	reqBody, err := buildChatCompletionsRequestBody(call)
	if err != nil {
		return LLMResult{}, err
	}

	payload, err := c.doChatCompletionsRequest(ctx, reqBody)
	if err != nil {
		c.Log.Warnf("openai request failed: %v", err)
		// Some servers/models (e.g., Ollama + certain models) may not support tools.
		// If we see a 400 with a clear message, retry once without tools.
		if len(call.Tools) > 0 && isToolsNotSupportedError(err) {
			c.Log.Infof("tools not supported; retrying without tools")
			callNoTools := call
			callNoTools.Tools = nil
			reqBody2, err2 := buildChatCompletionsRequestBody(callNoTools)
			if err2 != nil {
				return LLMResult{}, err
			}
			payload2, err2 := c.doChatCompletionsRequest(ctx, reqBody2)
			if err2 != nil {
				c.Log.Warnf("openai retry without tools failed: %v", err2)
				return LLMResult{}, err
			}
			return parseChatCompletionsResponse(payload2)
		}
		return LLMResult{}, err
	}

	return parseChatCompletionsResponse(payload)
}

func isToolsNotSupportedError(err error) bool {
	// Keep this string-based to avoid exposing HTTP error types in the interface.
	msg := err.Error()
	return strings.Contains(msg, "does not support tools")
}

func buildChatCompletionsRequestBody(call LLMRequest) (map[string]any, error) {
	msgs := make([]map[string]any, 0, len(call.Messages))
	for _, m := range call.Messages {
		obj := map[string]any{"role": m.Role, "content": m.Content}
		if m.Name != "" {
			obj["name"] = m.Name
		}
		msgs = append(msgs, obj)
	}

	toolObjs := make([]map[string]any, 0, len(call.Tools))
	for _, t := range call.Tools {
		var schema any
		if len(t.JSONSchema) > 0 {
			if err := json.Unmarshal(t.JSONSchema, &schema); err != nil {
				return nil, fmt.Errorf("openai: invalid tool JSONSchema for %q: %w", t.Name, err)
			}
		}
		toolObjs = append(toolObjs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  schema,
			},
		})
	}

	body := map[string]any{
		"model":    call.Model,
		"messages": msgs,
	}
	if len(toolObjs) > 0 {
		body["tools"] = toolObjs
		body["tool_choice"] = "auto"
	}
	return body, nil
}

func (c *OpenAICompletionClient) doChatCompletionsRequest(ctx context.Context, body map[string]any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("openai: status %d: %s", res.StatusCode, string(payload))
	}
	return payload, nil
}

func parseChatCompletionsResponse(payload []byte) (LLMResult, error) {
	// Minimal subset:
	// choices[0].message.content
	// choices[0].message.tool_calls[{id,type,function{name,arguments}}]
	var raw struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(payload, &raw); err != nil {
		return LLMResult{}, fmt.Errorf("openai: decode: %w", err)
	}
	if len(raw.Choices) == 0 {
		return LLMResult{}, fmt.Errorf("openai: empty choices")
	}

	msg := raw.Choices[0].Message
	out := LLMResult{Text: msg.Content}
	for _, tc := range msg.ToolCalls {
		if tc.Type != "function" {
			continue
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:            tc.ID,
			Name:          tc.Function.Name,
			ArgumentsJSON: []byte(tc.Function.Arguments),
		})
	}

	return out, nil
}
