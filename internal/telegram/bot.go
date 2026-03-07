package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// Bot — Telegram-бот для приёма сообщений и вызова ingestion.
type Bot struct {
	token    string
	ownerID  int64
	dataPath string
	ingester ingestion.Ingester
}

// NewBot создаёт бота.
func NewBot(token string, ownerID int64, dataPath string, ingester ingestion.Ingester) *Bot {
	return &Bot{token: token, ownerID: ownerID, dataPath: dataPath, ingester: ingester}
}

// Run запускает long polling.
func (b *Bot) Run(ctx context.Context) error {
	logger := clog.FromContext(ctx)
	logger.Info("telegram bot: started")
	defer logger.Info("telegram bot: stopped")

	baseURL := "https://api.telegram.org/bot" + b.token
	offset := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			updates, nextOffset, err := b.getUpdates(ctx, baseURL, offset)
			if err != nil {
				logger.Warn("poll failed", "error", err)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(5 * time.Second):
				}

				continue
			}
			offset = nextOffset
			for _, u := range updates {
				b.handleUpdate(ctx, u)
			}
		}
	}
}

func (b *Bot) getUpdates(ctx context.Context, baseURL string, offset int) ([]update, int, error) {
	reqURL := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30", baseURL, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, offset, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, offset, err
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		OK     bool     `json:"ok"`
		Result []update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, offset, err
	}
	if !result.OK {
		return nil, offset, errors.New("telegram API error")
	}

	nextOffset := offset
	for _, u := range result.Result {
		if u.UpdateID >= nextOffset {
			nextOffset = u.UpdateID + 1
		}
	}

	return result.Result, nextOffset, nil
}

type update struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Text    string `json:"text"`
		Caption string `json:"caption"` // для медиа: фото, видео, документ
		Chat    *struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From *struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"message"`
}

func (b *Bot) handleUpdate(ctx context.Context, u update) {
	if u.Message == nil {
		return
	}

	if u.Message.From == nil || (b.ownerID != 0 && u.Message.From.ID != b.ownerID) {
		clog.FromContext(ctx).Warn("telegram bot: unauthorized message ignored",
			"from_id", func() int64 {
				if u.Message.From != nil {
					return u.Message.From.ID
				}

				return 0
			}(),
		)

		return
	}

	text := u.Message.Text
	if text == "" {
		text = u.Message.Caption
	}
	if text == "" {
		clog.FromContext(ctx).Warn("telegram bot: empty message ignored")

		return
	}

	var chatID int64
	if u.Message.Chat != nil {
		chatID = u.Message.Chat.ID
	}

	if chatID != 0 {
		_ = b.sendMessage(ctx, chatID, "Принял, обрабатываю...")
	}

	node, err := b.ingester.IngestText(ctx, text)
	if err != nil {
		clog.Errorf(ctx, "ingest failed: %w", err)
		if chatID != 0 {
			_ = b.sendMessage(ctx, chatID, "Ошибка при сохранении: "+err.Error())
		}

		return
	}

	if chatID != 0 && node != nil {
		confirmation := b.buildConfirmation(node)
		_ = b.sendMessage(ctx, chatID, confirmation)
	}
}

func (b *Bot) buildConfirmation(node *kb.Node) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "✓ Сохранено: %s\n", node.Path)
	if t, ok := node.Metadata["type"].(string); ok && t != "" {
		fmt.Fprintf(&sb, "Тип: %s\n", t)
	}
	if kws, ok := node.Metadata["keywords"]; ok {
		switch v := kws.(type) {
		case []any:
			strs := make([]string, 0, len(v))
			for _, k := range v {
				if s, ok := k.(string); ok {
					strs = append(strs, s)
				}
			}
			if len(strs) > 0 {
				fmt.Fprintf(&sb, "Keywords: %s\n", strings.Join(strs, ", "))
			}
		case []string:
			if len(v) > 0 {
				fmt.Fprintf(&sb, "Keywords: %s\n", strings.Join(v, ", "))
			}
		}
	}

	return sb.String()
}

func (b *Bot) sendMessage(ctx context.Context, chatID int64, text string) error {
	baseURL := "https://api.telegram.org/bot" + b.token
	reqURL := baseURL + "/sendMessage"

	payload, err := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}
