package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/muonsoft/clog"
	"github.com/strider2038/knowledge-db/internal/ingestion"
)

// Bot — Telegram-бот для приёма сообщений и вызова ingestion.
type Bot struct {
	token    string
	dataPath string
	ingester ingestion.Ingester
}

// NewBot создаёт бота.
func NewBot(token, dataPath string, ingester ingestion.Ingester) *Bot {
	return &Bot{token: token, dataPath: dataPath, ingester: ingester}
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
				time.Sleep(5 * time.Second)
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
		return nil, offset, fmt.Errorf("telegram API error")
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
		Text string `json:"text"`
		From *struct {
			ID int64 `json:"id"`
		} `json:"from"`
	} `json:"message"`
}

func (b *Bot) handleUpdate(ctx context.Context, u update) {
	if u.Message == nil {
		return
	}
	text := u.Message.Text
	if text == "" {
		return
	}
	_, err := b.ingester.IngestText(ctx, text)
	if err != nil {
		clog.FromContext(ctx).Warn("ingest failed", "error", err)
	}
}
