package llm

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion/fetcher"
)

// LLMOrchestrator — интерфейс для обработки входного текста через LLM с function calling.
type LLMOrchestrator interface {
	Process(ctx context.Context, input ProcessInput) (*ProcessResult, error)
}

// ProcessInput — входные данные для LLM-оркестратора.
type ProcessInput struct {
	Text             string
	ExistingThemes   []string
	ExistingKeywords []string
}

// ProcessResult — результат обработки.
type ProcessResult struct {
	Keywords   []string
	Annotation string
	ThemePath  string
	Slug       string
	Type       string
	SourceURL  string
	SourceDate *time.Time
	Content    string
	Title      string
}

// responsesClient — внутренний интерфейс для тестирования.
type responsesClient interface {
	New(ctx context.Context, params responses.ResponseNewParams, opts ...option.RequestOption) (*responses.Response, error)
}

// OpenAIOrchestrator — реализация LLMOrchestrator через OpenAI Responses API.
type OpenAIOrchestrator struct {
	client         responsesClient
	model          string
	contentFetcher fetcher.ContentFetcher
}

// NewOpenAIOrchestrator создаёт OpenAIOrchestrator.
func NewOpenAIOrchestrator(apiKey, apiURL, model string, contentFetcher fetcher.ContentFetcher) *OpenAIOrchestrator {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if apiURL != "" {
		opts = append(opts, option.WithBaseURL(apiURL))
	}
	client := openai.NewClient(opts...)

	return &OpenAIOrchestrator{
		client:         &client.Responses,
		model:          model,
		contentFetcher: contentFetcher,
	}
}

// newOpenAIOrchestratorWithClient создаёт OpenAIOrchestrator с кастомным клиентом (для тестов).
func newOpenAIOrchestratorWithClient(client responsesClient, model string, contentFetcher fetcher.ContentFetcher) *OpenAIOrchestrator {
	return &OpenAIOrchestrator{
		client:         client,
		model:          model,
		contentFetcher: contentFetcher,
	}
}

// Process обрабатывает входной текст через LLM с function calling loop.
// LLM может вызывать fetch_url_content, fetch_url_meta, затем create_node.
//
// Используется stateless-подход: полная история диалога в каждом запросе.
// Это необходимо для OpenRouter и других провайдеров, которые не поддерживают
// previous_response_id или конвертируют его в формат Chat Completions некорректно
// (ошибка: "messages with role 'tool' must be a response to a preceding message with 'tool_calls'").
//
// Контент, полученный через fetch_url_content, кешируется в памяти и подставляется
// в результат напрямую — LLM не воспроизводит его в create_node, чтобы избежать усечения.
func (o *OpenAIOrchestrator) Process(ctx context.Context, input ProcessInput) (*ProcessResult, error) {
	clog.FromContext(ctx).Info("ingest: llm process start", "text_len", len(input.Text), "themes", len(input.ExistingThemes), "keywords", len(input.ExistingKeywords))

	instructions := buildSystemPrompt(input.ExistingThemes, input.ExistingKeywords)
	tools := buildTools()

	inputItems := responses.ResponseInputParam{
		responses.ResponseInputItemParamOfMessage(input.Text, responses.EasyInputMessageRoleUser),
	}

	// fetchCache хранит результаты fetch_url_content, чтобы избежать усечения контента LLM-ом.
	fetchCache := make(map[string]*fetcher.FetchResult)

	var totalTokens int64
	const maxIterations = 10
	for i := range maxIterations {
		iterStart := time.Now()
		clog.FromContext(ctx).Info("ingest: llm iteration", "iteration", i+1, "max", maxIterations)

		params := responses.ResponseNewParams{
			Model:        shared.ResponsesModel(o.model), //nolint:unconvert // required: type ResponsesModel string ≠ string
			Instructions: openai.String(instructions),
			Tools:        tools,
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: inputItems,
			},
		}

		resp, err := o.client.New(ctx, params)
		if err != nil {
			return nil, errors.Errorf("llm process: %w", err)
		}
		usage := resp.Usage
		totalTokens += usage.TotalTokens
		clog.FromContext(ctx).Info("ingest: llm response",
			"iteration", i+1,
			"output_items", len(resp.Output),
			"duration_ms", time.Since(iterStart).Milliseconds(),
			"input_tokens", usage.InputTokens,
			"output_tokens", usage.OutputTokens,
			"total_tokens", usage.TotalTokens)

		result, nextInputItems, err := o.processResponse(ctx, resp, inputItems, fetchCache)
		if err != nil {
			clog.FromContext(ctx).Info("ingest: llm process failed", "total_tokens", totalTokens)

			return nil, errors.Errorf("llm process: %w", err)
		}

		if result != nil {
			clog.FromContext(ctx).Info("ingest: llm process complete", "total_tokens", totalTokens)

			return result, nil
		}

		if len(nextInputItems) == 0 {
			clog.FromContext(ctx).Info("ingest: llm process failed (no create_node, no tool calls)", "total_tokens", totalTokens)

			return nil, errors.Errorf("llm process: no create_node call and no tool calls")
		}
		clog.FromContext(ctx).Info("ingest: llm tool calls executed, continuing", "iteration", i+1)

		inputItems = nextInputItems
	}

	clog.FromContext(ctx).Info("ingest: llm process failed (max iterations)", "total_tokens", totalTokens)

	return nil, errors.Errorf("llm process: max iterations exceeded")
}

func (o *OpenAIOrchestrator) processResponse(
	ctx context.Context,
	resp *responses.Response,
	inputItems responses.ResponseInputParam,
	fetchCache map[string]*fetcher.FetchResult,
) (*ProcessResult, responses.ResponseInputParam, error) {
	var functionCalls responses.ResponseInputParam
	var toolOutputs responses.ResponseInputParam

	for _, item := range resp.Output {
		if item.Type != "function_call" {
			continue
		}

		switch item.Name {
		case "create_node":
			result, err := parseCreateNodeArgs(item.Arguments)
			if err != nil {
				return nil, nil, errors.Errorf("parse create_node args: %w", err)
			}
			clog.FromContext(ctx).Info("ingest: llm create_node", "theme", result.ThemePath, "slug", result.Slug)

			// Если есть кешированный контент для source_url — используем его напрямую,
			// не полагаясь на то что LLM воспроизвёл контент без усечения.
			if result.SourceURL != "" {
				if cached, ok := fetchCache[result.SourceURL]; ok && cached.Content != "" {
					clog.FromContext(ctx).Info("ingest: using cached fetch content", "url", result.SourceURL, "content_len", len(cached.Content))
					result.Content = cached.Content
					if result.Title == "" && cached.Title != "" {
						result.Title = cached.Title
					}
				}
			}

			return result, nil, nil

		case "fetch_url_content":
			url := extractURLFromArgs(item.Arguments)
			clog.FromContext(ctx).Info("ingest: llm call fetch_url_content", "url", url)
			functionCalls = append(functionCalls, responses.ResponseInputItemParamOfFunctionCall(item.Arguments, item.CallID, item.Name))
			fetchStart := time.Now()
			output, fetchResult := o.executeFetchURLContent(ctx, item.Arguments)
			if fetchResult != nil && url != "" {
				fetchCache[url] = fetchResult
			}
			toolOutputs = append(toolOutputs, responses.ResponseInputItemParamOfFunctionCallOutput(item.CallID, output))
			clog.FromContext(ctx).Info("ingest: llm fetch_url_content done", "url", url, "duration_ms", time.Since(fetchStart).Milliseconds())

		case "fetch_url_meta":
			url := extractURLFromArgs(item.Arguments)
			clog.FromContext(ctx).Info("ingest: llm call fetch_url_meta", "url", url)
			functionCalls = append(functionCalls, responses.ResponseInputItemParamOfFunctionCall(item.Arguments, item.CallID, item.Name))
			metaStart := time.Now()
			output := executeFetchURLMeta(ctx, item.Arguments)
			toolOutputs = append(toolOutputs, responses.ResponseInputItemParamOfFunctionCallOutput(item.CallID, output))
			clog.FromContext(ctx).Info("ingest: llm fetch_url_meta done", "url", url, "duration_ms", time.Since(metaStart).Milliseconds())
		}
	}

	if len(functionCalls) == 0 {
		return nil, nil, nil
	}

	nextInputItems := make(responses.ResponseInputParam, 0, len(inputItems)+len(functionCalls)+len(toolOutputs))
	nextInputItems = append(nextInputItems, inputItems...)
	nextInputItems = append(nextInputItems, functionCalls...)
	nextInputItems = append(nextInputItems, toolOutputs...)

	return nil, nextInputItems, nil
}

// executeFetchURLContent выполняет fetch_url_content.
// Возвращает JSON-строку для LLM и полный FetchResult для кеша.
// LLM получает усечённый контент (первые 2000 символов) для анализа метаданных,
// полный контент сохраняется в кеше и используется при create_node напрямую.
func (o *OpenAIOrchestrator) executeFetchURLContent(ctx context.Context, argsJSON string) (string, *fetcher.FetchResult) {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return jsonError("invalid arguments: " + err.Error()), nil
	}

	result, err := o.contentFetcher.Fetch(ctx, args.URL)
	if err != nil {
		return jsonError("fetch failed: " + err.Error()), nil
	}

	// Передаём LLM только часть контента для определения метаданных (тема, ключевые слова).
	// Полный контент будет взят из кеша при вызове create_node.
	const previewLen = 2000
	preview := result.Content
	if len(preview) > previewLen {
		preview = preview[:previewLen] + "\n\n[...контент усечён для анализа, полная версия будет сохранена автоматически...]"
	}

	type fetchOutput struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Author  string `json:"author"`
	}
	out, err := json.Marshal(fetchOutput{Title: result.Title, Content: preview, Author: result.Author})
	if err != nil {
		return jsonError("marshal failed: " + err.Error()), nil
	}

	return string(out), result
}

func executeFetchURLMeta(ctx context.Context, argsJSON string) string {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return jsonError("invalid arguments: " + err.Error())
	}

	meta, err := fetcher.FetchURLMeta(ctx, args.URL)
	if err != nil {
		return jsonError("fetch meta failed: " + err.Error())
	}

	type metaOutput struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	out, err := json.Marshal(metaOutput{Title: meta.Title, Description: meta.Description})
	if err != nil {
		return jsonError("marshal failed: " + err.Error())
	}

	return string(out)
}

func parseCreateNodeArgs(argsJSON string) (*ProcessResult, error) {
	//nolint:tagliatelle // snake_case required: these are LLM function call arguments defined in the system prompt
	var args struct {
		Keywords   []string `json:"keywords"`
		Annotation string   `json:"annotation"`
		ThemePath  string   `json:"theme_path"`
		Slug       string   `json:"slug"`
		Type       string   `json:"type"`
		SourceURL  string   `json:"source_url"`
		SourceDate string   `json:"source_date"`
		Content    string   `json:"content"`
		Title      string   `json:"title"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, errors.Errorf("unmarshal: %w", err)
	}

	result := &ProcessResult{
		Keywords:   args.Keywords,
		Annotation: args.Annotation,
		ThemePath:  args.ThemePath,
		Slug:       args.Slug,
		Type:       args.Type,
		SourceURL:  args.SourceURL,
		Content:    unescapeNewlines(args.Content),
		Title:      args.Title,
	}

	if args.SourceDate != "" {
		for _, layout := range []string{"2006-01-02", time.RFC3339} {
			if t, err := time.Parse(layout, args.SourceDate); err == nil {
				result.SourceDate = &t

				break
			}
		}
	}

	return result, nil
}

func extractURLFromArgs(argsJSON string) string {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}

	return args.URL
}

// unescapeNewlines заменяет буквальные последовательности \n на настоящие символы
// переноса строки. Некоторые LLM двойно экранируют newlines в JSON аргументах
// function call, что приводит к появлению литеральных \n в сохранённом контенте.
func unescapeNewlines(s string) string {
	return strings.ReplaceAll(s, `\n`, "\n")
}

func jsonError(msg string) string {
	type errOutput struct {
		Error string `json:"error"`
	}
	out, err := json.Marshal(errOutput{Error: msg})
	if err != nil {
		return `{"error":"marshal failed"}`
	}

	return string(out)
}
