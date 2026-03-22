package tools


// Registry and in-memory implementation.

// Registry is a simple DI-friendly holder for tools.
// You can later replace this with a dynamic loader, MCP client pool, etc.
type Registry interface {
	List() []Tool
	Get(name string) (Tool, bool)
}

type MemoryRegistry struct {
	byName map[string]Tool
}

func NewMemoryRegistry(ts ...Tool) *MemoryRegistry {
	m := &MemoryRegistry{byName: map[string]Tool{}}
	for _, t := range ts {
		m.byName[t.Name()] = t
	}
	return m
}

func (m *MemoryRegistry) List() []Tool {
	out := make([]Tool, 0, len(m.byName))
	for _, t := range m.byName {
		out = append(out, t)
	}
	return out
}

func (m *MemoryRegistry) Get(name string) (Tool, bool) {
	t, ok := m.byName[name]
	return t, ok
}
