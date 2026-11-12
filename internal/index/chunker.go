package index

import (
	"strings"
)

// TextChunk — фрагмент тела статьи (заголовок секции и текст).
type TextChunk struct {
	Heading string
	Content string
}

// ChunkText разбивает текст статьи на фрагменты по заголовкам ##.
// Секции > 500 токенов режутся по параграфам; секции < 100 токенов мержатся со следующей.
// Фрагмент длиннее лимита embedding API режется ещё раз, в том числе текст без пробелов.
func ChunkText(body string) []TextChunk {
	if strings.TrimSpace(body) == "" {
		return nil
	}

	sections := splitByH2(body)
	var chunks []TextChunk

	for i := range sections {
		sec := sections[i]
		tokenCount := estimateTokens(sec.content)

		if tokenCount < 100 && len(sec.content) <= maxChunkContentBytes && i+1 < len(sections) {
			sections[i+1].content = sec.content + "\n\n" + sections[i+1].content
			if sections[i+1].heading == "" {
				sections[i+1].heading = sec.heading
			}

			continue
		}

		if tokenCount > 500 {
			subChunks := splitLargeSection(sec.heading, sec.content)
			chunks = append(chunks, subChunks...)
		} else {
			chunks = append(chunks, chunkWithinLimit(sec.heading, sec.content)...)
		}
	}

	return chunks
}

type section struct {
	heading string
	content string
}

func splitByH2(body string) []section {
	lines := strings.Split(body, "\n")
	var sections []section
	var current section

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if current.content != "" || current.heading != "" {
				sections = append(sections, current)
			}
			current = section{heading: strings.TrimPrefix(line, "## ")}
		} else if current.content != "" || strings.TrimSpace(line) != "" {
			current.content += line + "\n"
		}
	}

	if current.content != "" || current.heading != "" {
		sections = append(sections, current)
	}

	return sections
}

func splitLargeSection(heading, content string) []TextChunk {
	paragraphs := splitByParagraphs(content)
	var chunks []TextChunk
	var buf strings.Builder
	bufTokens := 0

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		chunks = append(chunks, TextChunk{Heading: heading, Content: strings.TrimSpace(buf.String())})
		buf.Reset()
		bufTokens = 0
	}

	for _, paragraph := range paragraphs {
		for _, part := range splitToChunkParts(paragraph) {
			partTokens := estimateTokens(part)
			separator := 0
			if buf.Len() > 0 {
				separator = 2
			}
			if buf.Len() > 0 && (bufTokens+partTokens > 500 || buf.Len()+separator+len(part) > maxChunkContentBytes) {
				flush()
			}
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(part)
			bufTokens += partTokens
		}
	}

	flush()

	return chunks
}

func chunkWithinLimit(heading, content string) []TextChunk {
	parts := splitToChunkParts(content)
	if len(parts) == 0 {
		return nil
	}

	chunks := make([]TextChunk, len(parts))
	for i, part := range parts {
		chunks[i] = TextChunk{Heading: heading, Content: part}
	}

	return chunks
}

// splitToChunkParts cuts text so each piece fits one embeddings input.
// A single oversized paragraph, including text with no spaces, is split too.
func splitToChunkParts(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if estimateTokens(text) <= 500 && len(text) <= maxChunkContentBytes {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) <= 1 {
		return splitByByteLimit(strings.TrimSpace(text), maxChunkContentBytes)
	}

	var parts []string
	var buf strings.Builder
	wordCount := 0
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		parts = append(parts, buf.String())
		buf.Reset()
		wordCount = 0
	}

	for _, word := range words {
		if len(word) > maxChunkContentBytes {
			flush()
			parts = append(parts, splitByByteLimit(word, maxChunkContentBytes)...)

			continue
		}

		extra := len(word)
		if buf.Len() > 0 {
			extra++
		}
		// estimateTokens counts words*1.3, so the budget is in words, not per-word estimates.
		if buf.Len() > 0 && (int(float64(wordCount+1)*1.3) > 500 || buf.Len()+extra > maxChunkContentBytes) {
			flush()
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(word)
		wordCount++
	}

	flush()

	return parts
}

func splitByParagraphs(content string) []string {
	var paragraphs []string
	var buf strings.Builder

	for line := range strings.SplitSeq(content, "\n") {
		if strings.TrimSpace(line) == "" {
			if buf.Len() > 0 {
				paragraphs = append(paragraphs, strings.TrimSpace(buf.String()))
				buf.Reset()
			}

			continue
		}
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(line)
	}

	if buf.Len() > 0 {
		paragraphs = append(paragraphs, strings.TrimSpace(buf.String()))
	}

	return paragraphs
}

func estimateTokens(text string) int {
	words := len(strings.Fields(text))

	return int(float64(words) * 1.3)
}
