package llm

import (
	"context"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"

	"github.com/muonsoft/errors"
)

const translateInstructions = `Переведи следующий markdown-текст на русский. Сохрани структуру: заголовки, списки, code blocks оставь без изменений. Верни только перевод, без пояснений.`

const generateTitleInstructions = `Придумай читаемый заголовок для текста. Верни только заголовок, без кавычек и пояснений.`

// TranslateToRussian переводит контент на русский через Responses API без tools.
func (o *OpenAIOrchestrator) TranslateToRussian(ctx context.Context, content string) (string, error) {
	params := responses.ResponseNewParams{
		Model:        shared.ResponsesModel(o.model), //nolint:unconvert // required: type ResponsesModel string ≠ string
		Instructions: openai.String(translateInstructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
			},
		},
	}

	resp, err := o.client.New(ctx, params)
	if err != nil {
		return "", errors.Errorf("translate: %w", err)
	}

	return resp.OutputText(), nil
}

// GenerateTitle генерирует читаемый заголовок для текста через Responses API без tools.
func (o *OpenAIOrchestrator) GenerateTitle(ctx context.Context, content string) (string, error) {
	params := responses.ResponseNewParams{
		Model:        shared.ResponsesModel(o.model), //nolint:unconvert // required: type ResponsesModel string ≠ string
		Instructions: openai.String(generateTitleInstructions),
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: responses.ResponseInputParam{
				responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
			},
		},
	}

	resp, err := o.client.New(ctx, params)
	if err != nil {
		return "", errors.Errorf("generate title: %w", err)
	}

	return strings.TrimSpace(resp.OutputText()), nil
}
