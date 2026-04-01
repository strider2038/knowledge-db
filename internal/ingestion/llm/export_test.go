package llm

import "github.com/strider2038/knowledge-db/internal/ingestion/fetcher"

// PickResourceURLFromMessageText экспортирует извлечение URL из текста для тестов в пакете llm_test.
var PickResourceURLFromMessageText = pickResourceURLFromMessageText

// NewTestOrchestrator creates an OpenAIOrchestrator with a custom client for testing.
func NewTestOrchestrator(client responsesClient, model string, contentFetcher fetcher.ContentFetcher) *OpenAIOrchestrator {
	return newOpenAIOrchestratorWithClient(client, model, contentFetcher)
}

// NewTestOrchestratorWithMetaFetcher creates an OpenAIOrchestrator with custom meta fetcher for testing.
func NewTestOrchestratorWithMetaFetcher(
	client responsesClient,
	model string,
	contentFetcher fetcher.ContentFetcher,
	metaFetcher fetcher.URLMetaFetcher,
) *OpenAIOrchestrator {
	return newOpenAIOrchestratorWithClientAndMetaFetcher(client, model, contentFetcher, metaFetcher)
}
