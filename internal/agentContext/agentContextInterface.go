package agentContext

// AgentContext is a session-scoped memory store.
//
// Design goals:
// - keep it LLM-agnostic (can be used by OpenAI, Anthropic, local models, etc.)
// - keep it transport-agnostic (can be persisted in-memory, Redis, DB, etc.)
// - support ReAct-style message history
type AgentContext interface {
	// Messages returns the chat history in a model-agnostic format.
	Messages() []Message
	// Append adds a message to the history.
	Append(msg Message)
	// Reset clears the history.
	Reset()
}

// Message is an LLM-agnostic chat message.
// Role is typically: "system" | "user" | "assistant" | "tool".
type Message struct {
	Role    string
	Content string
	// Name is optional (e.g., tool name).
	Name string
}
