//go:build ignore

package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/kyo/AIAgent/internal/agentContext"
)

// OpenAIClient is a minimal OpenAI Responses API client.
// It supports tool calling (function tools).
// Env:
// - OPENAI_API_KEY (required)
// - OPENAI_MODEL (default: gpt-4.1-mini)
// - OPENAI_BASE_URL (default: https://api.openai.com)
//
// Note: This is intentionally small and interface-driven; you can swap it out later.
type OpenAIClient struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Model      string
}

func NewOpenAIClientFromEnv() *OpenAIClient {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4.1-mini"
	}
	base := os.Getenv("OPENAI_BASE_URL")
	if base == "" {
		base = "https://api.openai.com"
	}
	return &OpenAIClient{
		HTTPClient: http.DefaultClient,
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		BaseURL:    base,
		Model:      model,
	}
}

func (c *OpenAIClient) ChatCompletion(ctx context.Context, messages []agentContext.Message, tools []ToolSpec) (LLMResponse, error) {
	if err := c.ensureDefaults(); err != nil {
		return LLMResponse{}, err
	}

	body, err := c.buildResponsesRequestBody(messages, tools)
	if err != nil {
		return LLMResponse{}, err
	}

	payload, err := c.doResponsesAPIRequest(ctx, body)
	if err != nil {
		return LLMResponse{}, err
	}

	return parseResponsesPayload(payload)
}

func (c *OpenAIClient) ensureDefaults() error {
	if c.APIKey == "" {
		return fmt.Errorf("openai: missing OPENAI_API_KEY")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	if c.Model == "" {
		c.Model = "gpt-4.1-mini"
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://api.openai.com"
	}
	return nil
}

func (c *OpenAIClient) buildResponsesRequestBody(messages []agentContext.Message, tools []ToolSpec) (map[string]any, error) {
	input := toResponsesAPIInput(messages)
	toolObjs, err := toResponsesAPITools(tools)
	if err != nil {
		return nil, err
	}

	body := map[string]any{
		"model": c.Model,
		"input": input,
	}
	if len(toolObjs) > 0 {
		body["tools"] = toolObjs
		body["tool_choice"] = "auto"
	}
	return body, nil
}

func toResponsesAPIInput(messages []agentContext.Message) []map[string]any {
	input := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		obj := map[string]any{"role": m.Role, "content": m.Content}
		if m.Name != "" {
			obj["name"] = m.Name
		}
		input = append(input, obj)
	}
	return input
}

func toResponsesAPITools(tools []ToolSpec) ([]map[string]any, error) {
	toolObjs := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
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
	return toolObjs, nil
}

func (c *OpenAIClient) doResponsesAPIRequest(ctx context.Context, body map[string]any) ([]byte, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/responses", bytes.NewReader(b))
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

func parseResponsesPayload(payload []byte) (LLMResponse, error) {
	// Parse minimal subset of Responses API.
	var raw struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			Name      string          `json:"name"`
			CallID    string          `json:"call_id"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"output"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return LLMResponse{}, fmt.Errorf("openai: decode: %w", err)
	}

	out := LLMResponse{}
	for _, o := range raw.Output {
		switch o.Type {
		case "message":
			for _, c := range o.Content {
				if c.Type == "output_text" {
					out.Text += c.Text
				}
			}
		case "function_call":
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:            o.CallID,
				Name:          o.Name,
				ArgumentsJSON: o.Arguments,
			})
		}
	}

	return out, nil
}
