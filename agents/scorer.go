package agents

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
	"github.com/ollama/ollama/api"
)

func NewLeadScorer(ctx context.Context, baseURL, model string) (*host.Specialist, error) {
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   model,
		Options: &api.Options{
			Temperature: 0.2,
		},
	})
	if err != nil {
		return nil, err
	}

	chain := compose.NewChain[[]*schema.Message, *schema.Message]()

	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		system := &schema.Message{
			Role: schema.System,
			Content: `You are a lead scoring agent. Based on the given lead details, score the quality of the lead from 1 to 10, and briefly explain why.
You are scoring based on these criteria:
- The lead’s title and decision-making power
- Company size (ideal: 50–500 employees)
- Industry (ideal: SaaS, B2B tech)
- Relevance to our product (enterprise tools)

Respond in this format:
Score: <number from 1 to 10>
Reason: <short reason>`,
		}
		return append([]*schema.Message{system}, input...), nil
	})).
		AppendChatModel(chatModel)

	r, err := chain.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "lead_scorer",
			IntendedUse: "Evaluate and score a lead based on relevance, size, title, and industry",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return r.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
		},
	}, nil
}
