package translation_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strider2038/knowledge-db/internal/ingestion/translation"
)

type mockTranslationClient struct {
	translateFunc func(ctx context.Context, content string) (string, error)
}

func (m *mockTranslationClient) TranslateToRussian(ctx context.Context, content string) (string, error) {
	if m.translateFunc != nil {
		return m.translateFunc(ctx, content)
	}

	return "translated: " + content, nil
}

func TestLLMTranslator_Translate_WhenShort_ExpectSingleCall(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	calls := 0
	mock := &mockTranslationClient{
		translateFunc: func(_ context.Context, content string) (string, error) {
			calls++

			return "ru: " + content, nil
		},
	}
	tr := translation.NewLLMTranslator(mock)

	result, err := tr.Translate(ctx, "Short English text for translation.")

	require.NoError(t, err)
	assert.Equal(t, "ru: Short English text for translation.", result)
	assert.Equal(t, 1, calls)
}

func TestLLMTranslator_Translate_WhenLong_ExpectChunked(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	paragraph := strings.Repeat("word ", 200)   // ~1000 chars
	text := strings.Repeat(paragraph+"\n\n", 8) // ~8000+ chars

	calls := 0
	mock := &mockTranslationClient{
		translateFunc: func(_ context.Context, content string) (string, error) {
			calls++

			return "ru: " + content[:min(50, len(content))], nil
		},
	}
	tr := translation.NewLLMTranslator(mock)

	result, err := tr.Translate(ctx, text)

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Greater(t, calls, 1)
}

var errTranslationFailed = errors.New("translation failed")

func TestLLMTranslator_Translate_WhenError_ExpectPropagated(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mock := &mockTranslationClient{
		translateFunc: func(_ context.Context, _ string) (string, error) {
			return "", errTranslationFailed
		},
	}
	tr := translation.NewLLMTranslator(mock)

	_, err := tr.Translate(ctx, "Some text")

	require.Error(t, err)
	assert.ErrorIs(t, err, errTranslationFailed)
}
