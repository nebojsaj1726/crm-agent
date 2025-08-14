package tools

import (
	"context"

	allPrompts "github.com/nebojsaj1726/crm-agent/prompts"
	"github.com/nebojsaj1726/crm-agent/utils"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

type FilterLeadsTool struct {
	LLM llms.Model
}

func (t FilterLeadsTool) Name() string {
	return "filter_leads"
}

func (t FilterLeadsTool) Description() string {
	return "Extract structured filters (company, department, title keywords) from a fuzzy lead description."
}

func (t FilterLeadsTool) Call(ctx context.Context, input string) (string, error) {
	return utils.RunPrompt(ctx, t.LLM, allPrompts.Filter, map[string]any{"input": input})
}

var _ tools.Tool = (*FilterLeadsTool)(nil)
