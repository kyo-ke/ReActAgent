package llmclient

import (
	"context"
	"encoding/json"
	"testing"

	"google.golang.org/genai"
)

func TestGemini_buildGeminiTools_JSONSchemaPassthrough(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}},"required":["a","b"]}`)

	tools, err := buildGeminiTools([]Tool{{
		Name:        "multiply",
		Description: "Multiply two integers",
		JSONSchema:  schema,
	}})
	if err != nil {
		t.Fatalf("buildGeminiTools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if len(tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("expected 1 function declaration, got %d", len(tools[0].FunctionDeclarations))
	}

	decl := tools[0].FunctionDeclarations[0]
	if decl.Name != "multiply" {
		t.Fatalf("unexpected name: %q", decl.Name)
	}
	if decl.Description == "" {
		t.Fatalf("expected description to be set")
	}
	if decl.ParametersJsonSchema == nil {
		t.Fatalf("expected ParametersJsonSchema to be set")
	}

	b, err := json.Marshal(decl.ParametersJsonSchema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	// Compare as JSON objects to avoid formatting diffs.
	var got, want any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal(schema, &want); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if string(mustJSON(t, got)) != string(mustJSON(t, want)) {
		t.Fatalf("schema mismatch\n got: %s\nwant: %s", mustJSON(t, got), mustJSON(t, want))
	}
}

func TestGemini_buildGeminiContentsFromMessages_ToolMessageToFunctionResponse(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "hi"}, {Role: "tool", Name: "multiply", Content: `{"value": 6}`}}
	contents, err := buildGeminiContentsFromMessages(msgs)
	if err != nil {
		t.Fatalf("buildGeminiContentsFromMessages: %v", err)
	}
	if len(contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(contents))
	}
	if contents[1].Role != genai.RoleUser {
		t.Fatalf("expected tool result content role to be RoleUser, got %q", contents[1].Role)
	}
	if len(contents[1].Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(contents[1].Parts))
	}
	if contents[1].Parts[0].FunctionResponse == nil {
		t.Fatalf("expected FunctionResponse part")
	}
	if contents[1].Parts[0].FunctionResponse.Name != "multiply" {
		t.Fatalf("unexpected FunctionResponse.Name: %q", contents[1].Parts[0].FunctionResponse.Name)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}

func TestGeminiClient_ensureDefaults_setsModel(t *testing.T) {
	gc := &GeminiClient{Client: &genai.Client{}, Model: "", Log: nil}
	req, err := gc.ensureDefaults(LLMRequest{})
	if err != nil {
		t.Fatalf("ensureDefaults: %v", err)
	}
	if req.Model == "" {
		t.Fatalf("expected req.Model to be populated")
	}
	if gc.Model == "" {
		t.Fatalf("expected client Model to be populated")
	}
}

// Compile-time interface assertion.
var _ LLMClient = (*GeminiClient)(nil)

// Avoid unused import warnings in case build tags change.
var _ = context.Background
