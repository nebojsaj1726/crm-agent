package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	allPrompts "github.com/nebojsaj1726/crm-agents/prompts"
	"github.com/nebojsaj1726/crm-agents/utils"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	model := os.Getenv("OLLAMA_MODEL")
	embedModel := os.Getenv("EMBED_MODEL")

	seedCmd := flag.Bool("seed", false, "Seed Chroma vector store from markdown")
	queryCmd := flag.Bool("query", false, "Query Chroma with user input")
	deleteCmd := flag.Bool("delete", false, "Delete all data from the Chroma vector store")
	flag.Parse()

	ctx := context.Background()

	if *seedCmd {
		err := utils.LoadMarkdownToVectorStore(ctx, "leads-example.md", "leads-demo")
		if err != nil {
			log.Fatalf("failed to seed vector DB: %v", err)
		}
		fmt.Println("Chroma vector store seeded successfully.")
		return
	}

	if *queryCmd {
		llm, err := ollama.New(ollama.WithModel(model))
		if err != nil {
			log.Fatalf("failed to create Ollama client: %v", err)
		}

		fmt.Print("Enter a fuzzy lead description: ")
		reader := bufio.NewReader(os.Stdin)
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		userInput = strings.TrimSpace(userInput)
		filterResp, err := utils.RunPrompt(ctx, llm, allPrompts.Filter, map[string]any{"input": userInput})
		if err != nil {
			log.Fatal(err)
		}
		var filters struct {
			Company       string   `json:"company"`
			Department    string   `json:"department"`
			TitleKeywords []string `json:"title_keywords"`
		}
		err = json.Unmarshal([]byte(filterResp), &filters)
		if err != nil {
			log.Fatalf("failed to parse JSON from LLM output: %v", err)
		}

		query := filters.Company + " " + filters.Department + " " + strings.Join(filters.TitleKeywords, " ")

		store, err := initVectorStore(embedModel)
		if err != nil {
			log.Fatalf("failed to initialize vector store: %v", err)
		}

		results, err := store.SimilaritySearch(ctx, query, 3)
		if err != nil {
			log.Fatalf("similarity search failed: %v", err)
		}

		const minScore = 0.6
		filtered := make([]schema.Document, 0)
		for _, r := range results {
			if r.Score >= minScore {
				filtered = append(filtered, r)
			}
		}

		if len(filtered) == 0 {
			fmt.Println("No highly relevant leads found.")
		} else {
			topLead := filtered[0]
			fmt.Printf("Top Lead (score: %.2f):\n%s\n\n", topLead.Score, topLead.PageContent)

			leadText := topLead.PageContent
			productDesc, err := utils.LoadMarkdownContent("product-example.md")
			if err != nil {
				log.Fatalf("failed to load product description: %v", err)
			}

			scoreCh := utils.RunPromptAsync(ctx, llm, allPrompts.Scoring, map[string]any{
				"lead":    leadText,
				"product": productDesc,
			})
			emailCh := utils.RunPromptAsync(ctx, llm, allPrompts.Email, map[string]any{
				"lead":    leadText,
				"product": productDesc,
			})

			scoreResult := <-scoreCh
			emailResult := <-emailCh

			if scoreResult.Err != nil {
				log.Printf("Error scoring lead: %v", scoreResult.Err)
			} else {
				fmt.Println("Lead Score & Justification:", scoreResult.Response)
			}

			if emailResult.Err != nil {
				log.Printf("Error generating email: %v", emailResult.Err)
			} else {
				fmt.Println("\nSuggested Prospecting Email:\n" + emailResult.Response)
			}
		}
		return
	}

	if *deleteCmd {
		store, err := initVectorStore(embedModel)
		if err != nil {
			log.Fatalf("failed to initialize vector store: %v", err)
		}

		err = store.RemoveCollection()
		if err != nil {
			log.Fatalf("failed to delete Chroma collection: %v", err)
		}
		fmt.Println("Chroma vector store deleted successfully.")
		return
	}

	fmt.Println("Please specify either -seed or -query")
}

func initVectorStore(embedModel string) (*chroma.Store, error) {
	llmEmbed, err := ollama.New(ollama.WithModel(embedModel))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama embed model: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(llmEmbed)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	store, err := chroma.New(
		chroma.WithChromaURL("http://localhost:8000"),
		chroma.WithEmbedder(embedder),
		chroma.WithDistanceFunction("cosine"),
		chroma.WithNameSpace("leads-demo"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open vector store: %w", err)
	}
	return &store, nil
}
