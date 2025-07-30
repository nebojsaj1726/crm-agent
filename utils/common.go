package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

func LoadMarkdownContent(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return string(data), nil
}

func extractKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func RunPrompt(ctx context.Context, llm llms.Model, templateStr string, input map[string]any) (string, error) {
	tmpl := prompts.NewPromptTemplate(templateStr, extractKeys(input))
	chain := chains.NewLLMChain(llm, tmpl)
	out, err := chain.Call(ctx, input)
	if err != nil {
		return "", err
	}
	return out["text"].(string), nil
}

type PromptResult struct {
	Response string
	Err      error
}

func RunPromptAsync(ctx context.Context, llm llms.Model, tmpl string, vars map[string]any) <-chan PromptResult {
	ch := make(chan PromptResult)
	go func() {
		resp, err := RunPrompt(ctx, llm, tmpl, vars)
		ch <- PromptResult{Response: resp, Err: err}
	}()
	return ch
}
