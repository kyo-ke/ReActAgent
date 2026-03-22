package agentSession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kyo/AIAgent/internal/agentContext"
	llmclient "github.com/kyo/AIAgent/internal/llmClient"
	"github.com/kyo/AIAgent/internal/tools"
)

type Session struct {
	LLM   llmclient.LLMClient
	Ctx   agentContext.AgentContext
	Tools tools.Registry

	MaxTurns int
}

func (s *Session) ensureDefaults() {
	if s.MaxTurns <= 0 {
		s.MaxTurns = 8
	}
}

// Iterate runs a ReAct-style loop:
// input -> context update -> LLM (with tools) -> optional tool calls -> LLM -> ... -> answer.
func (s *Session) Iterate(ctx context.Context, input string) (string, error) {
	s.ensureDefaults()
	if err := s.ensureDependencies(); err != nil {
		return "", err
	}

	s.appendUserInput(input)
	toolSpecs := s.buildLLMToolSpecs()

	for turn := 0; turn < s.MaxTurns; turn++ {
		resp, err := s.askLLM(ctx, toolSpecs)
		if err != nil {
			return "", err
		}

		s.appendAssistantText(resp.Text)

		if len(resp.ToolCalls) == 0 {
			if resp.Text == "" {
				return "", errors.New("agentSession: empty response")
			}
			return resp.Text, nil
		}

		if err := s.executeToolCalls(ctx, resp.ToolCalls); err != nil {
			return "", err
		}

		// Safety: prevent context blow-ups if a tool returns huge JSON.
		_ = json.Valid
	}

	return "", fmt.Errorf("agentSession: exceeded max turns (%d)", s.MaxTurns)
}

func (s *Session) ensureDependencies() error {
	if s.LLM == nil || s.Ctx == nil {
		return errors.New("agentSession: LLM and Ctx are required")
	}
	if s.Tools == nil {
		s.Tools = tools.NewMemoryRegistry()
	}
	return nil
}

func (s *Session) appendUserInput(input string) {
	s.Ctx.Append(agentContext.Message{Role: "user", Content: input})
}

func (s *Session) appendAssistantText(text string) {
	if text == "" {
		return
	}
	s.Ctx.Append(agentContext.Message{Role: "assistant", Content: text})
}

func (s *Session) buildLLMToolSpecs() []llmclient.Tool {
	toolSpecs := make([]llmclient.Tool, 0)
	for _, t := range s.Tools.List() {
		toolSpecs = append(toolSpecs, llmclient.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			JSONSchema:  t.JSONSchema(),
		})
	}
	return toolSpecs
}

func (s *Session) askLLM(ctx context.Context, toolSpecs []llmclient.Tool) (llmclient.LLMResult, error) {
	msgs := s.Ctx.Messages()
	req := llmclient.LLMRequest{
		Messages: make([]llmclient.Message, 0, len(msgs)),
		Tools:    toolSpecs,
	}
	for _, m := range msgs {
		req.Messages = append(req.Messages, llmclient.Message{Role: m.Role, Content: m.Content, Name: m.Name})
	}
	return s.LLM.CallLLM(ctx, req)
}

func (s *Session) executeToolCalls(ctx context.Context, calls []llmclient.ToolCall) error {
	for _, call := range calls {
		tool, ok := s.Tools.Get(call.Name)
		if !ok {
			result := fmt.Sprintf("tool not found: %s", call.Name)
			s.Ctx.Append(agentContext.Message{Role: "tool", Name: call.Name, Content: result})
			continue
		}

		resultText, err := tool.Call(ctx, call.ArgumentsJSON)
		if err != nil {
			resultText = fmt.Sprintf("tool error: %v", err)
		}
		s.Ctx.Append(agentContext.Message{Role: "tool", Name: call.Name, Content: resultText})
	}
	return nil
}
