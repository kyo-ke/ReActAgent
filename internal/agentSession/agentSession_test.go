package agentSession

import (
	"context"
	"testing"

	"github.com/kyo/AIAgent/internal/agentContext"
	llmclient "github.com/kyo/AIAgent/internal/llmClient"
	"github.com/kyo/AIAgent/internal/tools"
)

type fakeLLM struct {
	calls int
}

func (f *fakeLLM) CallLLM(ctx context.Context, call llmclient.LLMRequest) (llmclient.LLMResult, error) {
	f.calls++
	if f.calls == 1 {
		return llmclient.LLMResult{
			Text: "I'll call a tool.",
			ToolCalls: []llmclient.ToolCall{{
				Name:          "echo",
				ArgumentsJSON: []byte(`{"text":"hello"}`),
			}},
		}, nil
	}
	return llmclient.LLMResult{Text: "final: hello"}, nil
}

func TestSessionIterate_ToolLoop(t *testing.T) {
	s := &Session{
		LLM:   &fakeLLM{},
		Ctx:   agentContext.NewMemoryContext(),
		Tools: tools.NewMemoryRegistry(tools.EchoTool{}),
	}

	ans, err := s.Iterate(context.Background(), "say hi")
	if err != nil {
		t.Fatalf("Ask error: %v", err)
	}
	if ans != "final: hello" {
		t.Fatalf("unexpected answer: %q", ans)
	}
}
