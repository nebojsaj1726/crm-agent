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

func NewLeadEnricher(ctx context.Context, baseURL, model string) (*host.Specialist, error) {
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   model,
		Options: &api.Options{
			Temperature: 0.7,
		},
	})
	if err != nil {
		return nil, err
	}

	chain := compose.NewChain[[]*schema.Message, *schema.Message]()

	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		systemMsg := &schema.Message{
			Role: schema.System,
			Content: `You are an AI CRM assistant. Given a lead with basic data like name, email, and company name, enrich it by researching:
- Job title or role (e.g. CEO, Developer)
- Company description
- Industry
- Estimated company size
- Company website (guess)
Return the enriched lead as a JSON object.`,
		}
		return append([]*schema.Message{systemMsg}, input...), nil
	}))

	chain.AppendChatModel(chatModel)

	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*schema.Message, error) {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "Enriched Lead:\n" + msg.Content,
		}, nil
	}))

	r, err := chain.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "lead_enricher",
			IntendedUse: "Enrich basic lead info with company details and role guesses",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return r.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
		},
	}, nil
}
