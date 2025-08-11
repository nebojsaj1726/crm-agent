package tools

import (
	"context"
	"encoding/json"

	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

type VectorSearchTool struct {
	Store *chroma.Store
}

func (t VectorSearchTool) Name() string {
	return "vector_search"
}

func (t VectorSearchTool) Description() string {
	return "Search the Chroma vector store for leads matching a description. Returns JSON array of {score, lead_text}."
}

func (t VectorSearchTool) Call(ctx context.Context, input string) (string, error) {
	results, err := t.Store.SimilaritySearch(ctx, input, 3)
	if err != nil {
		return "", err
	}
	var leads []map[string]any
	for _, r := range results {
		leads = append(leads, map[string]any{
			"score":     r.Score,
			"lead_text": r.PageContent,
		})
	}
	b, _ := json.Marshal(leads)
	return string(b), nil
}

var _ tools.Tool = (*VectorSearchTool)(nil)
