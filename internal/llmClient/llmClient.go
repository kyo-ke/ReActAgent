package llmclient

import "context"

// LLMClient is intentionally minimal.
// This layer only knows how to execute a single function call against some LLM backend.
//
// The agent (ReAct loop) lives one level above and can be swapped without changing the LLM backend.
type LLMClient interface {
	CallLLM(ctx context.Context, req LLMRequest) (LLMResult, error)
}

// LLMRequest is an LLM-agnostic description of a single call:
// provide chat messages + optional function tools, and let the backend return either text or tool calls.
type LLMRequest struct {
	Model string
	// Messages is either already formatted for the backend, or a simplified schema.
	// We keep it generic to avoid coupling this package to a specific message type.
	Messages []Message
	Tools    []Tool
}

type Message struct {
	Role    string
	Content string
	Name    string
}

type Tool struct {
	Name        string
	Description string
	// JSONSchema is JSON Schema for the tool arguments.
	JSONSchema []byte
}

type ToolCall struct {
	ID            string
	Name          string
	ArgumentsJSON []byte
}

type LLMResult struct {
	Text      string
	ToolCalls []ToolCall
}
