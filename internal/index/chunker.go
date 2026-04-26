package index

import (
	"strings"
)

// Chunk представляет фрагмент тела статьи.
type TextChunk struct {
	Heading string
	Content string
}

// ChunkText разбивает текст статьи на фрагменты по заголовкам ##.
// Секции > 500 токенов режутся по параграфам; секции < 100 токенов мержатся со следующей.
func ChunkText(body string) []TextChunk {
	if strings.TrimSpace(body) == "" {
		return nil
	}

	sections := splitByH2(body)
	var chunks []TextChunk

	for i := 0; i < len(sections); i++ {
		sec := sections[i]
		tokenCount := estimateTokens(sec.content)

		if tokenCount < 100 && i+1 < len(sections) {
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
			chunks = append(chunks, TextChunk{Heading: sec.heading, Content: sec.content})
		}
	}

	return chunks
}

type section struct {
	heading  string
	content  string
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
		} else {
			if current.content != "" || strings.TrimSpace(line) != "" {
				current.content += line + "\n"
			}
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
	bufLen := 0

	for _, p := range paragraphs {
		pTokens := estimateTokens(p)
		if bufLen+pTokens > 500 && bufLen > 0 {
			chunks = append(chunks, TextChunk{Heading: heading, Content: strings.TrimSpace(buf.String())})
			buf.Reset()
			bufLen = 0
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(p)
		bufLen += pTokens
	}

	if buf.Len() > 0 {
		chunks = append(chunks, TextChunk{Heading: heading, Content: strings.TrimSpace(buf.String())})
	}

	return chunks
}

func splitByParagraphs(content string) []string {
	var paragraphs []string
	var buf strings.Builder

	for _, line := range strings.Split(content, "\n") {
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
