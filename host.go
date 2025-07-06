package main

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
)

func newHost(ctx context.Context, baseURL, modelName string) (*host.Host, error) {
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, err
	}

	return &host.Host{
		ChatModel: chatModel,
		SystemPrompt: `
You have three internal tools: lead_enricher, lead_scorer, email_writer.
Whenever the user submits a lead, you must:
1. Call lead_enricher.
2. Immediately call lead_scorer on that output.
3. Finally call email_writer.
Return **only** the message from email_writer to the user; hide all intermediate outputs.
`,
	}, nil
}
