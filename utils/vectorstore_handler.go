package utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

func saveToVectorStore(ctx context.Context, docs []schema.Document, namespace string) error {
	llm, err := ollama.New(ollama.WithModel("nomic-embed-text:v1.5"))
	if err != nil {
		return fmt.Errorf("failed to create ollama client: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return fmt.Errorf("failed to create embedder: %w", err)
	}

	store, err := chroma.New(
		chroma.WithChromaURL("http://localhost:8000"),
		chroma.WithEmbedder(embedder),
		chroma.WithDistanceFunction("cosine"),
		chroma.WithNameSpace(namespace),
	)
	if err != nil {
		return fmt.Errorf("failed to create chroma store: %w", err)
	}

	filteredDocs := make([]schema.Document, 0, len(docs))
	for _, doc := range docs {
		if doc.PageContent != "" {
			filteredDocs = append(filteredDocs, doc)
		}
	}

	_, err = store.AddDocuments(ctx, filteredDocs)
	if err != nil {
		slog.Warn("Error adding documents", "error", err)
		return fmt.Errorf("error adding documents: %w", err)
	}

	return nil
}

func LoadMarkdownToVectorStore(ctx context.Context, path string, namespace string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read markdown file: %w", err)
	}

	vectorLoader := documentloaders.NewText(strings.NewReader(string(content)))

	splitter := textsplitter.NewMarkdownTextSplitter(
		textsplitter.WithSeparators([]string{"---"}),
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(0),
	)

	docs, err := vectorLoader.LoadAndSplit(ctx, splitter)
	if err != nil {
		return fmt.Errorf("failed to split document: %w", err)
	}

	for i := range docs {
		docs[i].Metadata = map[string]any{
			"source": path,
		}
		fmt.Printf("Chunk %d:\n%s\n---\n", i+1, docs[i].PageContent)
	}

	return saveToVectorStore(ctx, docs, namespace)
}

func LoadMarkdownContent(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return string(data), nil
}

func ExtractKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
