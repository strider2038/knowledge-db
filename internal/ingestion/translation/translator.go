package translation

import (
	"context"
	"strings"
	"time"

	"github.com/muonsoft/clog"
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
	start := time.Now()
	textWithoutBlocks, blocks := extractCodeBlocks(content)
	textWithoutBlocks = strings.TrimSpace(textWithoutBlocks)

	clog.Info(ctx, "translate: start",
		"input_len", len(content),
		"text_len", len(textWithoutBlocks),
		"code_blocks", len(blocks),
	)

	if len(textWithoutBlocks) <= chunkThreshold {
		translated, err := t.client.TranslateToRussian(ctx, textWithoutBlocks)
		if err != nil {
			return "", err
		}

		clog.Info(ctx, "translate: complete",
			"chunked", false,
			"duration_ms", time.Since(start).Milliseconds(),
		)

		return reinsertCodeBlocks(translated, blocks), nil
	}

	chunks := splitIntoChunks(textWithoutBlocks)
	translatedChunks := make([]string, 0, len(chunks))

	clog.Info(ctx, "translate: chunking", "chunks", len(chunks))

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
			clog.Errorf(ctx, "translate chunk %d/%d: %w", i+1, len(chunks), err)

			return "", err
		}
		translatedChunks = append(translatedChunks, translated)

		clog.Info(ctx, "translate: chunk done",
			"chunk", i+1,
			"total_chunks", len(chunks),
		)
	}

	merged := mergeChunks(translatedChunks)

	clog.Info(ctx, "translate: complete",
		"chunked", true,
		"chunks", len(chunks),
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return reinsertCodeBlocks(merged, blocks), nil
}
