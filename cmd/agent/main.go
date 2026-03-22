package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/kyo/AIAgent/internal/agentContext"
	"github.com/kyo/AIAgent/internal/agentSession"
	llmclient "github.com/kyo/AIAgent/internal/llmClient"
	"github.com/kyo/AIAgent/internal/tools"
)

func main() {
	ctx := context.Background()

	llm := llmclient.NewOpenAICompletionClientFromEnv()
	mem := agentContext.NewMemoryContext()
	reg := tools.NewMemoryRegistry(tools.EchoTool{})

	s := &agentSession.Session{
		LLM:      llm,
		Ctx:      mem,
		Tools:    reg,
		MaxTurns: 8,
	}

	in := bufio.NewScanner(os.Stdin)
	fmt.Println("Ask me something (Ctrl-D to exit):")
	for {
		fmt.Print("> ")
		if !in.Scan() {
			break
		}
		q := in.Text()
		if q == "" {
			continue
		}
		ans, err := s.Iterate(ctx, q)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		fmt.Println(ans)
	}
}
