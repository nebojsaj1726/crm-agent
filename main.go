package main

import (
	"bufio"
	"context"
	"io"
	"os"

	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
	"github.com/nebojsaj1726/crm-agents/agents"
)

func main() {
	_ = godotenv.Load(".env")

	ollamaBaseURL := os.Getenv("OLLAMA_BASE_URL")
	ollamaModel := os.Getenv("OLLAMA_MODEL")

	ctx := context.Background()

	h, err := newHost(ctx, ollamaBaseURL, ollamaModel)
	if err != nil {
		panic(err)
	}

	enricher, err := agents.NewLeadEnricher(ctx, ollamaBaseURL, ollamaModel)
	if err != nil {
		panic(err)
	}

	scorer, err := agents.NewLeadScorer(ctx, ollamaBaseURL, ollamaModel)
	if err != nil {
		panic(err)
	}

	writer, err := agents.NewEmailWriter(ctx, ollamaBaseURL, ollamaModel)
	if err != nil {
		panic(err)
	}

	hostMA, err := host.NewMultiAgent(ctx, &host.MultiAgentConfig{
		Host: *h,
		Specialists: []*host.Specialist{
			enricher,
			scorer,
			writer,
		},
	})
	if err != nil {
		panic(err)
	}

	cb := &logCallback{}

	for {
		println("\nType lead info or 'exit'")

		var message string
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			message += scanner.Text()
			break
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}

		if message == "exit" {
			return
		}

		msg := &schema.Message{
			Role:    schema.User,
			Content: message,
		}

		out, err := hostMA.Stream(ctx, []*schema.Message{msg}, host.WithAgentCallbacks(cb))
		if err != nil {
			panic(err)
		}

		println("\nAnswer:")

		for {
			msg, err := out.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			print(msg.Content)
		}

		out.Close()
	}
}

type logCallback struct{}

func (l *logCallback) OnHandOff(ctx context.Context, info *host.HandOffInfo) context.Context {
	println("\n[Agent Handoff] ➤", info.ToAgentName, "with input:", info.Argument)
	return ctx
}
