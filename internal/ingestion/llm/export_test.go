package llm

import "github.com/strider2038/knowledge-db/internal/ingestion/fetcher"

// NewTestOrchestrator creates an OpenAIOrchestrator with a custom client for testing.
func NewTestOrchestrator(client responsesClient, model string, contentFetcher fetcher.ContentFetcher) *OpenAIOrchestrator {
	return newOpenAIOrchestratorWithClient(client, model, contentFetcher)
}
