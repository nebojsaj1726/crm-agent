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
	"time"

	"github.com/nebojsaj1726/crm-agents/utils"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

const scoringPrompt = `You are an expert B2B sales assistant.

You are helping qualify leads for the following product:

{{.product}}

Given the lead information below, assign a lead score from 1 to 10 based on likelihood to convert. Also provide a one-line justification.

Respond ONLY with a JSON object like: {"score": 8, "justification": "strong procurement pain points and large team"}

Lead Information:
{{.lead}}`

const emailPrompt = `You are a prospecting assistant helping write short, personalized cold emails.

You are reaching out to a lead about the following product:

{{.product}}

Given the lead information below, generate a 3-sentence email that:
- Acknowledges the lead's role
- References their pain points
- Explains clearly how the product can help

Respond ONLY with the email text (no JSON, no labels).

Lead Information:
{{.lead}}`

const filterPrompt = `You are an expert assistant that extracts structured filters from fuzzy lead descriptions.

Given an input description, extract and return a JSON object with the following fields:
- "company": The company name mentioned (string, or null if missing)
- "department": The department or team (string, or null if missing)
- "title_keywords": A list of keywords describing the job title (e.g., ["buyer", "manager"], or empty list if none)

Respond ONLY with the JSON object.

Input: "{{.input}}"`

func main() {
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
		llm, err := ollama.New(ollama.WithModel("llama3.1:8b"))
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
		startfilter := time.Now()
		filterResp, err := runPrompt(ctx, llm, filterPrompt, map[string]any{"input": userInput})
		fmt.Println("filtering took:", time.Since(startfilter))
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

		store, err := initVectorStore()
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

			leadText := buildLeadString(topLead)
			productDesc, err := utils.LoadMarkdownContent("product-example.md")
			if err != nil {
				log.Fatalf("failed to load product description: %v", err)
			}

			scoreCh := make(chan string)
			scoreErrCh := make(chan error)
			emailCh := make(chan string)
			emailErrCh := make(chan error)

			go func() {
				startscore := time.Now()
				scoreResp, err := runPrompt(ctx, llm, scoringPrompt, map[string]any{
					"lead":    leadText,
					"product": productDesc,
				})
				fmt.Println("scoring took:", time.Since(startscore))
				scoreCh <- scoreResp
				scoreErrCh <- err
			}()

			go func() {
				startemail := time.Now()
				emailResp, err := runPrompt(ctx, llm, emailPrompt, map[string]any{
					"lead":    leadText,
					"product": productDesc,
				})
				fmt.Println("emailing took:", time.Since(startemail))
				emailCh <- emailResp
				emailErrCh <- err
			}()

			scoreResp := <-scoreCh
			scoreErr := <-scoreErrCh
			emailResp := <-emailCh
			emailErr := <-emailErrCh

			if scoreErr != nil {
				log.Printf("Error scoring lead: %v", scoreErr)
			} else {
				fmt.Println("Lead Score & Justification:", scoreResp)
			}
			if emailErr != nil {
				log.Printf("Error generating email: %v", emailErr)
			} else {
				fmt.Println("\nSuggested Prospecting Email:\n" + emailResp)
			}
		}
		return
	}

	if *deleteCmd {
		store, err := initVectorStore()
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

func initVectorStore() (*chroma.Store, error) {
	llmEmbed, err := ollama.New(ollama.WithModel("nomic-embed-text:v1.5"))
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

func runPrompt(ctx context.Context, llm llms.Model, templateStr string, input map[string]any) (string, error) {
	tmpl := prompts.NewPromptTemplate(templateStr, utils.ExtractKeys(input))
	chain := chains.NewLLMChain(llm, tmpl)
	out, err := chain.Call(ctx, input)
	if err != nil {
		return "", err
	}
	return out["text"].(string), nil
}

func buildLeadString(doc schema.Document) string {
	return doc.PageContent
}
