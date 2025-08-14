package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/nebojsaj1726/crm-agent/tools"
	"github.com/nebojsaj1726/crm-agent/utils"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

type Lead struct {
	Score    float64 `json:"score"`
	LeadText string  `json:"lead_text"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	model := os.Getenv("OLLAMA_MODEL")
	embedModel := os.Getenv("EMBED_MODEL")

	seedCmd := flag.Bool("seed", false, "Seed Chroma vector store from markdown")
	queryCmd := flag.Bool("query", false, "Query the system with user input")
	deleteCmd := flag.Bool("delete", false, "Delete all data from the Chroma vector store")
	webCmd := flag.Bool("web", false, "Run web")

	flag.Parse()

	ctx := context.Background()

	llm, err := ollama.New(ollama.WithModel(model))
	if err != nil {
		log.Fatalf("failed to create Ollama client: %v", err)
	}

	store, err := initVectorStore(embedModel)
	if err != nil {
		log.Fatalf("failed to initialize vector store: %v", err)
	}

	filterTool := tools.FilterLeadsTool{LLM: llm}
	vectorTool := tools.VectorSearchTool{Store: store}
	scoringTool := tools.ScoreLeadTool{LLM: llm}
	emailTool := tools.DraftEmailTool{LLM: llm}

	switch {
	case *seedCmd:
		err := utils.LoadMarkdownToVectorStore(ctx, "leads-example.md", "leads-demo")
		if err != nil {
			log.Fatalf("failed to seed vector DB: %v", err)
		}
		log.Println("Chroma vector store seeded successfully.")

	case *queryCmd:
		fmt.Print("Enter a fuzzy lead description: ")
		reader := bufio.NewReader(os.Stdin)
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		userInput = strings.TrimSpace(userInput)

		resp, err := runQuery(ctx, userInput, filterTool, vectorTool, scoringTool, emailTool)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Top Lead (Vector search score: %.2f):\n%s\n\n", resp.TopLead.Score, resp.TopLead.LeadText)
		fmt.Println("Lead Score & Justification:", resp.LeadScore)
		fmt.Println("\nSuggested Prospecting Email:\n" + resp.ProspectEmail)

	case *deleteCmd:
		store, err := initVectorStore(embedModel)
		if err != nil {
			log.Fatalf("failed to initialize vector store: %v", err)
		}

		err = store.RemoveCollection()
		if err != nil {
			log.Fatalf("failed to delete Chroma collection: %v", err)
		}
		log.Println("Chroma vector store deleted successfully.")

	case *webCmd:
		startAPI(ctx, filterTool, vectorTool, scoringTool, emailTool)

	default:
		fmt.Println("Please enter command: -seed | -query | -delete | -web")
	}

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
