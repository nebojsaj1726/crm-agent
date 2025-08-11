package tools

import (
	"context"
	"fmt"

	allPrompts "github.com/nebojsaj1726/crm-agents/prompts"
	"github.com/nebojsaj1726/crm-agents/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

type ScoreLeadTool struct {
	LLM llms.Model
}

func (t ScoreLeadTool) Name() string {
	return "score_lead"
}

func (t ScoreLeadTool) Description() string {
	return "Given a lead and product description, returns JSON score and justification."
}

func (t ScoreLeadTool) Call(ctx context.Context, input string) (string, error) {
	productDesc, err := utils.LoadMarkdownContent("product-example.md")
	if err != nil {
		return "", fmt.Errorf("failed to load product description: %w", err)
	}
	return utils.RunPrompt(ctx, t.LLM, allPrompts.Scoring, map[string]any{
		"lead":    input,
		"product": productDesc,
	})
}

type DraftEmailTool struct {
	LLM llms.Model
}

func (t DraftEmailTool) Name() string {
	return "draft_email"
}

func (t DraftEmailTool) Description() string {
	return "Given a lead and product description, returns a short 3-sentence prospecting email."
}

func (t DraftEmailTool) Call(ctx context.Context, input string) (string, error) {
	productDesc, err := utils.LoadMarkdownContent("product-example.md")
	if err != nil {
		return "", fmt.Errorf("failed to load product description: %w", err)
	}
	return utils.RunPrompt(ctx, t.LLM, allPrompts.Email, map[string]any{
		"lead":    input,
		"product": productDesc,
	})
}

var _ tools.Tool = (*ScoreLeadTool)(nil)
var _ tools.Tool = (*DraftEmailTool)(nil)
