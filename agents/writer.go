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

func NewEmailWriter(ctx context.Context, baseURL, model string) (*host.Specialist, error) {
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
		system := &schema.Message{
			Role: schema.System,
			Content: `You are a helpful sales assistant. 
Given an enriched lead profile, your job is to draft a professional, engaging, and brief email to initiate contact with the lead.`,
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
			Name:        "write_email",
			IntendedUse: "take enriched lead profile and write initial outreach email",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return r.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
		},
	}, nil
}
