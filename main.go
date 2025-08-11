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
	"github.com/nebojsaj1726/crm-agents/tools"
	"github.com/nebojsaj1726/crm-agents/utils"
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

		store, err := initVectorStore(embedModel)
		if err != nil {
			log.Fatalf("failed to initialize vector store: %v", err)
		}

		filterTool := tools.FilterLeadsTool{LLM: llm}
		vectorTool := tools.VectorSearchTool{Store: store}
		scoringTool := tools.ScoreLeadTool{LLM: llm}
		emailTool := tools.DraftEmailTool{LLM: llm}

		fmt.Print("Enter a fuzzy lead description: ")
		reader := bufio.NewReader(os.Stdin)
		userInput, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		userInput = strings.TrimSpace(userInput)

		filterResp, err := filterTool.Call(ctx, userInput)
		if err != nil {
			log.Fatal(err)
		}
		var filters struct {
			Company       string   `json:"company"`
			Department    string   `json:"department"`
			TitleKeywords []string `json:"title_keywords"`
		}
		if err := json.Unmarshal([]byte(filterResp), &filters); err != nil {
			log.Fatalf("failed to parse JSON from filter tool: %v", err)
		}
		query := strings.TrimSpace(filters.Company + " " + filters.Department + " " + strings.Join(filters.TitleKeywords, " "))

		searchResp, err := vectorTool.Call(ctx, query)
		if err != nil {
			log.Fatal(err)
		}
		var searchResults []Lead
		if err := json.Unmarshal([]byte(searchResp), &searchResults); err != nil {
			log.Fatalf("failed to parse JSON from vector tool: %v", err)
		}

		const minScore = 0.6
		var topLead *Lead
		for i := range searchResults {
			if searchResults[i].Score >= minScore {
				if topLead == nil || searchResults[i].Score > topLead.Score {
					topLead = &searchResults[i]
				}
			}
		}
		if topLead == nil {
			fmt.Println("No highly relevant leads found.")
			return
		}

		fmt.Printf("Top Lead (Vector serch score: %.2f):\n%s\n\n", topLead.Score, topLead.LeadText)

		scoreResp, err := scoringTool.Call(ctx, topLead.LeadText)
		if err != nil {
			log.Printf("Error scoring lead: %v", err)
		} else {
			fmt.Println("Lead Score & Justification:", scoreResp)
		}

		emailResp, err := emailTool.Call(ctx, topLead.LeadText)
		if err != nil {
			log.Printf("Error generating email: %v", err)
		} else {
			fmt.Println("\nSuggested Prospecting Email:\n" + emailResp)
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
