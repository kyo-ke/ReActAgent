package tools

// This file intentionally contains small convenience helpers and package-level
// exports. It must not be empty, otherwise the Go compiler fails the package.

// NewRegistry is kept for backward compatibility with earlier refactors.
// Prefer constructing a concrete registry like MemoryRegistry via
// NewMemoryRegistry.
func NewRegistry(ts ...Tool) *MemoryRegistry {
	return NewMemoryRegistry(ts...)
}
