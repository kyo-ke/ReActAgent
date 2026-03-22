package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/kyo/AIAgent/internal/logging"
	"google.golang.org/genai"
)

// GeminiClient implements LLMClient using google.golang.org/genai (Gemini API).
//
// Env:
// - GEMINI_API_KEY (required)
// - GEMINI_MODEL (default: gemini-2.0-flash)
//
// Notes about tools/function calling:
// - LLMRequest.Tools are mapped to genai.Tool{FunctionDeclarations: ...}
// - Model function calls are returned via GenerateContentResponse.FunctionCalls()
//   and mapped to LLMResult.ToolCalls.
// - Tool results are expected to be provided back to the model as messages with
//   Role="tool" and Name=function name (agentSession already does this). go-genai
//   uses FunctionResponse parts; this client translates Role="tool" messages into
//   FunctionResponse parts when building the next request.
//
// This client only supports text + function calling (no multimodal).
//
// Ref: https://pkg.go.dev/google.golang.org/genai
//
// IMPORTANT: this uses the *Gemini API* backend (API key). Vertex AI is not
// configured here.

type GeminiClient struct {
	Client *genai.Client
	Model  string
	Log    logging.Logger
}

func NewGeminiClientFromEnvWithLogger(log logging.Logger) (*GeminiClient, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("gemini: missing GEMINI_API_KEY")
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}
	if log == nil {
		log = logging.New()
	}

	c, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: key})
	if err != nil {
		return nil, fmt.Errorf("gemini: new client: %w", err)
	}

	return &GeminiClient{Client: c, Model: model, Log: log}, nil
}

func (c *GeminiClient) ensureDefaults(req LLMRequest) (LLMRequest, error) {
	if c.Client == nil {
		return LLMRequest{}, fmt.Errorf("gemini: nil Client")
	}
	if c.Log == nil {
		c.Log = logging.New()
	}
	if c.Model == "" {
		c.Model = "gemini-2.0-flash"
	}
	if req.Model == "" {
		req.Model = c.Model
	}
	return req, nil
}

func (c *GeminiClient) CallLLM(ctx context.Context, req LLMRequest) (LLMResult, error) {
	req, err := c.ensureDefaults(req)
	if err != nil {
		return LLMResult{}, err
	}

	contents, err := buildGeminiContentsFromMessages(req.Messages)
	if err != nil {
		return LLMResult{}, err
	}

	tools, err := buildGeminiTools(req.Tools)
	if err != nil {
		return LLMResult{}, err
	}

	cfg := &genai.GenerateContentConfig{}
	if len(tools) > 0 {
		cfg.Tools = tools
		cfg.ToolConfig = &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeAuto}}
	}

	c.Log.Debugf("gemini generateContent: model=%s messages=%d tools=%d", req.Model, len(req.Messages), len(req.Tools))
	resp, err := c.Client.Models.GenerateContent(ctx, req.Model, contents, cfg)
	if err != nil {
		return LLMResult{}, fmt.Errorf("gemini: generate content: %w", err)
	}

	out := LLMResult{Text: resp.Text()}

	for _, fc := range resp.FunctionCalls() {
		argsJSON, err := json.Marshal(fc.Args)
		if err != nil {
			return LLMResult{}, fmt.Errorf("gemini: marshal function call args for %q: %w", fc.Name, err)
		}
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:            fc.ID,
			Name:          fc.Name,
			ArgumentsJSON: argsJSON,
		})
	}

	return out, nil
}

func buildGeminiTools(ts []Tool) ([]*genai.Tool, error) {
	if len(ts) == 0 {
		return nil, nil
	}
	decls := make([]*genai.FunctionDeclaration, 0, len(ts))
	for _, t := range ts {
		var schema any
		if len(t.JSONSchema) > 0 {
			if err := json.Unmarshal(t.JSONSchema, &schema); err != nil {
				return nil, fmt.Errorf("gemini: invalid tool JSONSchema for %q: %w", t.Name, err)
			}
		}

		decls = append(decls, &genai.FunctionDeclaration{
			Name:                 t.Name,
			Description:          t.Description,
			ParametersJsonSchema: schema,
		})
	}

	return []*genai.Tool{{FunctionDeclarations: decls}}, nil
}

func buildGeminiContentsFromMessages(msgs []Message) ([]*genai.Content, error) {
	contents := make([]*genai.Content, 0, len(msgs))

	for _, m := range msgs {
		switch m.Role {
		case "system":
			// Gemini has a distinct SystemInstruction field in config, but our LLMRequest
			// doesn’t expose it. For now, treat system messages as user text prefix.
			// This keeps backward compatibility with the existing agent design.
			contents = append(contents, genai.NewContentFromText(m.Content, genai.RoleUser))
		case "user":
			contents = append(contents, genai.NewContentFromText(m.Content, genai.RoleUser))
		case "assistant":
			contents = append(contents, genai.NewContentFromText(m.Content, genai.RoleModel))
		case "tool":
			// agentSession encodes tool results as {Role:"tool", Name:<tool>, Content:<resultText>}.
			// Gemini expects a FunctionResponse part.
			if m.Name == "" {
				// If we can’t map it, fall back to plain text to avoid dropping context.
				contents = append(contents, genai.NewContentFromText(m.Content, genai.RoleUser))
				continue
			}

			var responseObj map[string]any
			// If tool returns JSON, keep it structured; otherwise wrap in {"result": "..."}.
			if err := json.Unmarshal([]byte(m.Content), &responseObj); err != nil {
				responseObj = map[string]any{"result": m.Content}
			}
			contents = append(contents, genai.NewContentFromFunctionResponse(m.Name, responseObj, genai.RoleUser))
		default:
			return nil, fmt.Errorf("gemini: unsupported message role: %q", m.Role)
		}
	}

	return contents, nil
}
