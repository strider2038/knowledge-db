package git

import (
	"context"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"

	"github.com/muonsoft/clog"
)

const fallbackCommitMessage = "chore: manual commit via UI"

// commitResponsesClient — интерфейс для OpenAI Responses API (для тестирования).
type commitResponsesClient interface {
	New(ctx context.Context, params responses.ResponseNewParams, opts ...option.RequestOption) (*responses.Response, error)
}

// CommitMessageGenerator генерирует conventional commit message на основе git diff.
type CommitMessageGenerator struct {
	client commitResponsesClient
	model  string
}

// NewCommitMessageGenerator создаёт генератор commit message.
// Если apiKey пуст — Generate возвращает fallback.
func NewCommitMessageGenerator(apiKey, apiURL, model string) *CommitMessageGenerator {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if apiURL != "" {
		opts = append(opts, option.WithBaseURL(apiURL))
	}
	client := openai.NewClient(opts...)

	return &CommitMessageGenerator{
		client: &client.Responses,
		model:  model,
	}
}

// Generate генерирует commit message на основе diff stat.
// При ошибке LLM возвращает fallback-сообщение.
func (g *CommitMessageGenerator) Generate(ctx context.Context, diffStat string) string {
	if g == nil || g.client == nil {
		clog.Info(ctx, "commit message generator: not configured, using fallback")

		return fallbackCommitMessage
	}

	instructions := `Ты — помощник для git-коммитов. На основе git diff --stat сгенерируй один conventional commit message на английском.
Правила:
- Формат: type(scope): description
- Типы: feat, fix, docs, refactor, chore, style, test
- scope — опционально
- description — короткая сводка изменений (не более 72 символов)
- Ответ должен содержать ТОЛЬКО commit message, без пояснений
- Если изменений много — обобщай`

	input := "git diff --stat:\n\n" + diffStat

	params := responses.ResponseNewParams{
		Model:        g.model,
		Instructions: openai.String(instructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(input, responses.EasyInputMessageRoleUser),
			},
		},
	}

	resp, err := g.client.New(ctx, params)
	if err != nil {
		clog.Warn(ctx, "commit message generator: LLM error, using fallback", "error", err)

		return fallbackCommitMessage
	}

	message := extractCommitTextOutput(resp)
	message = strings.TrimSpace(message)
	if message == "" {
		clog.Warn(ctx, "commit message generator: empty LLM response, using fallback")

		return fallbackCommitMessage
	}

	if idx := strings.Index(message, "\n"); idx >= 0 {
		message = message[:idx]
	}

	return message
}

func extractCommitTextOutput(resp *responses.Response) string {
	for _, item := range resp.Output {
		if item.Type == "message" {
			for _, c := range item.Content {
				if c.Type == "output_text" {
					return c.Text
				}
			}
		}
	}

	return ""
}
