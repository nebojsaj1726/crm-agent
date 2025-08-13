package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/nebojsaj1726/crm-agents/tools"
)

type QueryResponse struct {
	TopLead       *Lead  `json:"top_lead"`
	LeadScore     string `json:"lead_score"`
	ProspectEmail string `json:"prospect_email"`
}

func runQuery(
	ctx context.Context,
	userInput string,
	filterTool tools.FilterLeadsTool,
	vectorTool tools.VectorSearchTool,
	scoringTool tools.ScoreLeadTool,
	emailTool tools.DraftEmailTool,
) (QueryResponse, error) {
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
		return QueryResponse{}, fmt.Errorf("no highly relevant leads found")
	}

	scoreResp, err := scoringTool.Call(ctx, topLead.LeadText)
	if err != nil {
		scoreResp = fmt.Sprintf("Error scoring lead: %v", err)
	}

	emailResp, err := emailTool.Call(ctx, topLead.LeadText)
	if err != nil {
		emailResp = fmt.Sprintf("Error generating email: %v", err)
	}

	return QueryResponse{
		TopLead:       topLead,
		LeadScore:     scoreResp,
		ProspectEmail: emailResp,
	}, nil
}
