package tools

import "context"

// Tool is the common contract for anything the agent can call.
// It intentionally uses JSON bytes so the agent stays resilient to schema changes.
//
// Implementations can wrap:
// - MCP calls (stdio/http)
// - in-process skills
// - external processes, etc.
type Tool interface {
	Name() string
	Description() string
	// JSONSchema returns a JSON Schema object describing the expected args.
	JSONSchema() []byte
	// Call executes the tool.
	Call(ctx context.Context, argumentsJSON []byte) (resultText string, err error)
}
