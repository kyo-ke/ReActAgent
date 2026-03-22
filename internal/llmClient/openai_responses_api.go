//go:build ignore

package llmclient

import (
	"fmt"
	"os"

	"github.com/kyo/AIAgent/internal/logging"
)

// This file is intentionally a skeleton for the OpenAI Responses API client.
//
// Notes:
// - It is build-ignored so it can't accidentally affect the main build.
// - Keep constructors here so wiring can be added later without changing env contracts.
// - The active OpenAI-compatible client used by the agent is openai_completion_client.go.
//
// If you later want to enable this client, remove the build tag and implement LLMClient.

// OpenAIResponsesClient is a placeholder for a future Responses API implementation.
// When implemented, it should satisfy LLMClient.
type OpenAIResponsesClient struct {
	Log logging.Logger
}

// NewOpenAIResponsesClientFromEnv constructs a Responses-API client from env vars.
//
// This is a stub while Responses API support is intentionally disabled.
func NewOpenAIResponsesClientFromEnv(log logging.Logger) (*OpenAIResponsesClient, error) {
	// Read vars so the intended contract is explicit.
	_ = os.Getenv("OPENAI_API_KEY")
	_ = os.Getenv("OPENAI_MODEL")
	_ = os.Getenv("OPENAI_BASE_URL")

	if log == nil {
		log = logging.New()
	}
	return &OpenAIResponsesClient{Log: log}, fmt.Errorf("openai responses client is not implemented (file is build-ignored)")
}
