package telegram

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/muonsoft/errors"
)

// ImportItem — извлечённая запись из сообщения Telegram для импорта.
type ImportItem struct {
	ID           int64  `json:"id"`
	DateUnixTime string `json:"date_unixtime"`
	Text         string `json:"text"`
	SourceAuthor string `json:"source_author"`
	SourceURL    string `json:"source_url"`
}

type chatExport struct {
	ID       int64           `json:"id"`
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Messages []messageExport `json:"messages"`
}

type messageExport struct {
	ID            int64           `json:"id"`
	Type          string          `json:"type"`
	DateUnixTime  string          `json:"date_unixtime"`
	From          string          `json:"from"`
	Text          json.RawMessage `json:"text"`
	TextEntities  []textEntity    `json:"text_entities"`
	Caption       json.RawMessage `json:"caption"`
	ForwardedFrom string          `json:"forwarded_from"`
	SavedFrom     string          `json:"saved_from"`
}

type textEntity struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Href string `json:"href"`
	URL  string `json:"url"`
}

// ParseChat парсит JSON экспорта одного чата Telegram.
// Возвращает записи, отсортированные по date_unixtime по убыванию (новые первыми).
func ParseChat(jsonData []byte) ([]ImportItem, error) {
	var chat chatExport
	if err := json.Unmarshal(jsonData, &chat); err != nil {
		return nil, errors.Errorf("parse telegram chat: %w", err)
	}

	var items []ImportItem
	for _, msg := range chat.Messages {
		if msg.Type != "message" {
			continue
		}

		text, sourceURL := extractTextAndURL(&msg)
		if text == "" {
			continue
		}

		sourceAuthor := msg.ForwardedFrom
		if sourceAuthor == "" {
			sourceAuthor = msg.SavedFrom
		}
		if sourceAuthor == "" {
			sourceAuthor = msg.From
		}

		items = append(items, ImportItem{
			ID:           msg.ID,
			DateUnixTime: msg.DateUnixTime,
			Text:         text,
			SourceAuthor: sourceAuthor,
			SourceURL:    sourceURL,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return compareUnixTime(items[i].DateUnixTime, items[j].DateUnixTime) > 0
	})

	return items, nil
}

func extractTextAndURL(msg *messageExport) (string, string) {
	var text, sourceURL string
	if len(msg.Text) > 0 {
		text, sourceURL = parseTextOrEntities(msg.Text, msg.TextEntities)
		if text != "" {
			return text, sourceURL
		}
	}
	if len(msg.Caption) > 0 {
		text, sourceURL = parseTextOrEntities(msg.Caption, nil)
	}

	return text, sourceURL
}

func parseTextOrEntities(raw json.RawMessage, entities []textEntity) (string, string) {
	if len(entities) > 0 {
		return parseFromEntities(entities)
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, ""
	}

	return parseFromTextArray(raw)
}

func parseFromEntities(entities []textEntity) (string, string) {
	var parts []string
	var firstLink string
	for _, e := range entities {
		t := e.Text
		if strings.TrimSpace(t) == "" {
			continue
		}
		part, url := formatEntity(e)
		parts = append(parts, part)
		if url != "" && firstLink == "" {
			firstLink = url
		}
	}

	return strings.Join(parts, ""), firstLink
}

func formatEntity(e textEntity) (string, string) {
	switch e.Type {
	case "plain":
		return e.Text, ""
	case "link":
		return "[" + e.Text + "](" + e.Text + ")", e.Text
	case "text_link":
		url := e.Href
		if url == "" {
			url = e.URL
		}
		if url != "" {
			return "[" + e.Text + "](" + url + ")", url
		}

		return e.Text, ""
	default:
		return e.Text, ""
	}
}

func parseFromTextArray(raw json.RawMessage) (string, string) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return "", ""
	}

	var parts []string
	var firstLink string
	for _, item := range arr {
		part, url := parseTextArrayItem(item)
		if part != "" {
			parts = append(parts, part)
			if url != "" && firstLink == "" {
				firstLink = url
			}
		}
	}

	return strings.Join(parts, ""), firstLink
}

func parseTextArrayItem(item json.RawMessage) (string, string) {
	var str string
	if err := json.Unmarshal(item, &str); err == nil {
		if looksLikeURL(str) {
			return str, str
		}

		return str, ""
	}

	var obj struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Href string `json:"href"`
		URL  string `json:"url"`
	}
	if err := json.Unmarshal(item, &obj); err != nil {
		return "", ""
	}
	if strings.TrimSpace(obj.Text) == "" {
		return "", ""
	}

	return formatEntity(textEntity{Type: obj.Type, Text: obj.Text, Href: obj.Href, URL: obj.URL})
}

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func compareUnixTime(a, b string) int {
	ai, _ := strconv.ParseInt(a, 10, 64)
	bi, _ := strconv.ParseInt(b, 10, 64)
	if ai < bi {
		return -1
	}
	if ai > bi {
		return 1
	}

	return 0
}
