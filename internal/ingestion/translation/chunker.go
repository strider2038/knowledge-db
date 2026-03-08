package translation

import (
	"fmt"
	"strings"
)

const (
	chunkThreshold  = 6000
	chunkTarget     = 4000
	overlapCheckLen = 80
)

// codeBlockPlaceholder — placeholder для подстановки code block при склейке.
type codeBlockPlaceholder struct {
	placeholder string
	content     string
}

// extractCodeBlocks удаляет code blocks из текста, сохраняет их в буфер.
// Возвращает текст с плейсхолдерами и слайс блоков для последующей вставки.
func extractCodeBlocks(content string) (string, []codeBlockPlaceholder) {
	var blocks []codeBlockPlaceholder
	text := codeBlockRe.ReplaceAllStringFunc(content, func(match string) string {
		idx := len(blocks)
		ph := fmt.Sprintf("___KB_CODE_%d___", idx)
		blocks = append(blocks, codeBlockPlaceholder{placeholder: ph, content: match})

		return ph
	})

	return text, blocks
}

// reinsertCodeBlocks заменяет плейсхолдеры на оригинальное содержимое code blocks.
func reinsertCodeBlocks(text string, blocks []codeBlockPlaceholder) string {
	for _, b := range blocks {
		text = strings.ReplaceAll(text, b.placeholder, b.content)
	}

	return text
}

// splitIntoChunks разбивает текст на чанки по границам абзацев.
// Целевой размер ~4000 символов, порог разбиения 6000.
func splitIntoChunks(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) <= chunkThreshold {
		return []string{text}
	}

	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, p := range paragraphs {
		if current.Len()+len(p)+2 > chunkTarget && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		// Если абзац сам длиннее порога — режем по \n
		if len(p) > chunkThreshold {
			lines := strings.SplitSeq(p, "\n")
			for line := range lines {
				if current.Len()+len(line)+1 > chunkTarget && current.Len() > 0 {
					chunks = append(chunks, strings.TrimSpace(current.String()))
					current.Reset()
				}
				if current.Len() > 0 {
					current.WriteString("\n")
				}
				current.WriteString(line)
			}
		} else {
			current.WriteString(p)
		}
	}
	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

// mergeChunks склеивает переводы чанков, удаляя дублирование на стыках.
func mergeChunks(chunks []string) string {
	if len(chunks) == 0 {
		return ""
	}
	if len(chunks) == 1 {
		return chunks[0]
	}

	var sb strings.Builder
	sb.WriteString(chunks[0])
	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1]
		curr := chunks[i]
		overlap := findOverlap(prev, curr)
		if overlap != "" {
			curr = strings.TrimPrefix(curr, overlap)
		}
		sb.WriteString("\n\n")
		sb.WriteString(curr)
	}

	return sb.String()
}

// findOverlap ищет дублирование: конец prev совпадает с началом curr.
func findOverlap(prev, curr string) string {
	prevLen := len(prev)
	currLen := len(curr)
	checkLen := overlapCheckLen
	if prevLen < checkLen || currLen < checkLen {
		checkLen = min(prevLen, currLen)
	}
	if checkLen == 0 {
		return ""
	}
	for n := checkLen; n >= 10; n-- {
		prevSuffix := prev[prevLen-n:]
		currPrefix := curr[:min(n, currLen)]
		if prevSuffix == currPrefix {
			return currPrefix
		}
	}

	return ""
}
