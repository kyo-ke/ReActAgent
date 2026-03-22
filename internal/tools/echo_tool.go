package tools

import (
	"context"
	"encoding/json"
)

type EchoTool struct{}

func (t EchoTool) Name() string        { return "echo" }
func (t EchoTool) Description() string { return "Echo back the input text." }

func (t EchoTool) JSONSchema() []byte {
	// {"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}
	b, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{"type": "string"},
		},
		"required": []string{"text"},
	})
	return b
}

func (t EchoTool) Call(ctx context.Context, argumentsJSON []byte) (string, error) {
	var args struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(argumentsJSON, &args); err != nil {
		return "", err
	}
	return args.Text, nil
}
