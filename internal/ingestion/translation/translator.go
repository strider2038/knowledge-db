package translation

import (
	"context"
	"strings"
)

// TranslationClient — интерфейс для вызова перевода через LLM.
type TranslationClient interface {
	TranslateToRussian(ctx context.Context, content string) (string, error)
}

// Translator — интерфейс перевода контента на русский.
type Translator interface {
	Translate(ctx context.Context, content string) (string, error)
}

// LLMTranslator — реализация Translator через LLM с поддержкой чанкинга.
type LLMTranslator struct {
	client TranslationClient
}

// NewLLMTranslator создаёт LLMTranslator.
func NewLLMTranslator(client TranslationClient) *LLMTranslator {
	return &LLMTranslator{client: client}
}

// Translate переводит контент на русский. При длине >6000 символов — чанкинг.
func (t *LLMTranslator) Translate(ctx context.Context, content string) (string, error) {
	textWithoutBlocks, blocks := extractCodeBlocks(content)
	textWithoutBlocks = strings.TrimSpace(textWithoutBlocks)

	if len(textWithoutBlocks) <= chunkThreshold {
		translated, err := t.client.TranslateToRussian(ctx, textWithoutBlocks)
		if err != nil {
			return "", err
		}

		return reinsertCodeBlocks(translated, blocks), nil
	}

	chunks := splitIntoChunks(textWithoutBlocks)
	translatedChunks := make([]string, 0, len(chunks))

	for i, chunk := range chunks {
		var input string
		if i == 0 {
			input = chunk
		} else {
			prevEnd := translatedChunks[i-1]
			suffix := prevEnd
			if len(suffix) > 200 {
				suffix = "..." + suffix[len(suffix)-150:]
			}
			input = "Это продолжение перевода. Предыдущая часть заканчивалась на: \"" + suffix + "\"\n\nПереведи следующий фрагмент, сохраняя связность.\n\n" + chunk
		}
		translated, err := t.client.TranslateToRussian(ctx, input)
		if err != nil {
			return "", err
		}
		translatedChunks = append(translatedChunks, translated)
	}

	merged := mergeChunks(translatedChunks)

	return reinsertCodeBlocks(merged, blocks), nil
}
