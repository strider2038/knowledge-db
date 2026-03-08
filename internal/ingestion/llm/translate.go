package llm

import (
	"context"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"

	"github.com/muonsoft/errors"
)

const translateInstructions = `Переведи следующий markdown-текст на русский. Сохрани структуру: заголовки, списки, code blocks оставь без изменений. Верни только перевод, без пояснений.`

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
