package index

import "unicode/utf8"

// maxEmbeddingInputBytes is the largest single embeddings input, in bytes.
// OpenAI-compatible APIs reject one input above 8192 tokens. A byte cap stays
// under that limit even when the tokenizer spends one token per byte.
const maxEmbeddingInputBytes = 8000

// maxChunkContentBytes leaves room for the heading prefix added before embed.
const maxChunkContentBytes = 7000

// clipEmbeddingInput shortens text to the embeddings API input limit.
// The cut is on a UTF-8 rune boundary. The second result is true when text was shortened.
func clipEmbeddingInput(text string) (string, bool) {
	if len(text) <= maxEmbeddingInputBytes {
		return text, false
	}

	parts := splitByByteLimit(text, maxEmbeddingInputBytes)
	if len(parts) == 0 {
		return "", true
	}

	return parts[0], true
}

func splitByByteLimit(text string, limit int) []string {
	if text == "" || limit <= 0 {
		return nil
	}

	var parts []string
	for len(text) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		if cut == 0 {
			_, size := utf8.DecodeRuneInString(text)
			if size <= 0 {
				break
			}
			cut = size
		}
		parts = append(parts, text[:cut])
		text = text[cut:]
	}

	if text != "" {
		parts = append(parts, text)
	}

	return parts
}
