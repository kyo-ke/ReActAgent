package llmclient

import (
	"fmt"
	"os"
	"strings"

	"github.com/kyo/AIAgent/internal/logging"
)

// NewClientFromEnv constructs an LLMClient based on environment variables.
//
// Env:
// - LLM_PROVIDER (optional): "completion" (default) or "gemini".
//
// Provider-specific env vars are validated by the chosen constructor.
func NewClientFromEnv(log logging.Logger) (LLMClient, error) {
	p := strings.TrimSpace(strings.ToLower(os.Getenv("LLM_PROVIDER")))
	if p == "" {
		p = "openai-completion"
	}

	switch p {
	case "openai-completion":
		return NewOpenAICompletionClientFromEnvWithLogger(log), nil
	case "openai-responses":
		// NOTE: Responses API client is currently build-ignored (see openai_responses_api.go).
		// Wire-up is kept here as a placeholder so env contracts don't change later.
		return nil, fmt.Errorf("llmclient: LLM_PROVIDER=%q is not available in this build (openai_responses_api is build-ignored)", p)
	case "gemini":
		return NewGeminiClientFromEnvWithLogger(log)
	default:
		return nil, fmt.Errorf("llmclient: unsupported LLM_PROVIDER %q (supported: openai-completion, openai-responses, gemini)", p)
	}
}
