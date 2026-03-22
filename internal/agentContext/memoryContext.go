package agentContext

import "sync"

// MemoryContext is a simple thread-safe in-memory implementation of AgentContext.
// Later you can swap it with Redis/DB without changing agent code.
type MemoryContext struct {
	mu   sync.Mutex
	msgs []Message
}

func NewMemoryContext() *MemoryContext {
	return &MemoryContext{}
}

func (m *MemoryContext) Messages() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Message, len(m.msgs))
	copy(out, m.msgs)
	return out
}

func (m *MemoryContext) Append(msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
}

func (m *MemoryContext) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = nil
}
