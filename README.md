# Agent implementation for self learning

## Behaviour
- Agent should remember the context
- Agent should handle multiple sessions
- Agent should able to use tools to interact with other server


## Implementation
- ReAct model agent implementation
- Agent will get information via stdio/http
- tools should be MCP server or SKILL
- Agent should be able to use multiple type of LLM model

## Quickstart (CLI)

This repo is written in Go and includes a minimal ReAct-style agent loop:

- `agentSession.Session` holds `LLMClient`, `AgentContext`, and `tools.Registry`
- `LLMClient` is interface-based (OpenAI implementation included)
- tools are interface-based (can be MCP / Skill)

### Requirements

- Go 1.22+
- OpenAI API key

### Environment variables

- `OPENAI_API_KEY` (required)
- `OPENAI_MODEL` (optional, default: `gpt-4.1-mini`)
- `OPENAI_BASE_URL` (optional, default: `https://api.openai.com`)

### Run

```zsh
export OPENAI_API_KEY="..."
go run ./cmd/agent
```

## Using Ollama (local OpenAI-compatible server)

If you have Ollama running locally (default: `http://localhost:11434`), you can point this agent to it via `OPENAI_BASE_URL`.

This project uses the **OpenAI Chat Completions API** (`POST /v1/chat/completions`). Ollama provides an OpenAI-compatible endpoint at:

- `http://localhost:11434/v1/chat/completions`

### 1) Check available models

```zsh
curl -sS http://localhost:11434/api/tags
```

Pick a model name from the response (for example: `gemma3:latest`).

### 2) Run the agent against Ollama

```zsh
export OPENAI_BASE_URL=http://localhost:11434
export OPENAI_API_KEY=ollama
export OPENAI_MODEL='gemma3:latest'
go run ./cmd/agent
```

### Notes about tools

Some Ollama models don't support tools / function calling. In that case, the client will automatically retry once without tools.

