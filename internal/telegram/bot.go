package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

const (
	forwardBufferTTL    = 3 * time.Second
	forwardFlushTimeout = 90 * time.Second
	maxPendingPerChat   = 10
)

// message — структура сообщения Telegram API (подмножество полей).
type message struct {
	MessageID      int             `json:"message_id"`
	Text           string          `json:"text"`
	Caption        string          `json:"caption"`
	ReplyToMessage *message        `json:"reply_to_message"`
	ForwardOrigin  json.RawMessage `json:"forward_origin,omitempty"`
	Chat           *struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	From *struct {
		ID int64 `json:"id"`
	} `json:"from"`
}

// Bot — Telegram-бот для приёма сообщений и вызова ingestion.
type Bot struct {
	token    string
	ownerID  int64
	ingester ingestion.Ingester
	buffer   *forwardBuffer
}

// NewBot создаёт бота.
func NewBot(token string, ownerID int64, ingester ingestion.Ingester) *Bot {
	b := &Bot{
		token:    token,
		ownerID:  ownerID,
		ingester: ingester,
	}
	b.buffer = newForwardBuffer(b)

	return b
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

// extractText извлекает text или caption из сообщения.
func extractText(msg *message) string {
	if msg == nil {
		return ""
	}
	if msg.Text != "" {
		return msg.Text
	}

	return msg.Caption
}

// combineForwardWithComment объединяет комментарий и пересланный контент с явными метками для LLM.
func combineForwardWithComment(comment, forwarded string) string {
	return "Инструкции пользователя: " + comment + "\nПересланное сообщение: " + forwarded
}

type bufferKey struct {
	chatID    int64
	messageID int
}

type pendingForward struct {
	text   string
	chatID int64
	timer  *time.Timer
}

type pendingComment struct {
	text  string
	timer *time.Timer
}

type forwardBuffer struct {
	mu       sync.Mutex
	pending  map[bufferKey]*pendingForward
	byChat   map[int64][]bufferKey     // для ограничения по chat
	comments map[int64]*pendingComment // chatID -> ожидающий комментарий пользователя
	bot      *Bot
	ttl      time.Duration // TTL для обоих буферов; переопределяется в тестах
}

func newForwardBuffer(b *Bot) *forwardBuffer {
	return &forwardBuffer{
		pending:  make(map[bufferKey]*pendingForward),
		byChat:   make(map[int64][]bufferKey),
		comments: make(map[int64]*pendingComment),
		bot:      b,
		ttl:      forwardBufferTTL,
	}
}

// addComment буферизует текстовый комментарий пользователя. Если после него придёт
// пересланное сообщение, они будут объединены. Иначе через TTL обрабатывается как standalone.
//
//nolint:contextcheck // timer callback runs async
func (fb *forwardBuffer) addComment(_ context.Context, chatID int64, text string) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if prev, ok := fb.comments[chatID]; ok {
		prev.timer.Stop()
	}
	pc := &pendingComment{text: text}
	pc.timer = time.AfterFunc(fb.ttl, func() {
		fb.mu.Lock()
		c, ok := fb.comments[chatID]
		if !ok || c != pc {
			fb.mu.Unlock()

			return
		}
		delete(fb.comments, chatID)
		fb.mu.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), forwardFlushTimeout)
		defer cancel()
		fb.bot.processIngest(ctx, text, chatID)
	})
	fb.comments[chatID] = pc
}

// takeComment извлекает ожидающий комментарий для чата (если есть), отменяя его таймер.
func (fb *forwardBuffer) takeComment(chatID int64) string {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	if pc, ok := fb.comments[chatID]; ok {
		pc.timer.Stop()
		delete(fb.comments, chatID)

		return pc.text
	}

	return ""
}

//nolint:contextcheck,unparam // timer callback runs async; ctx kept for API consistency
func (fb *forwardBuffer) add(ctx context.Context, chatID int64, messageID int, text string) bool {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	key := bufferKey{chatID: chatID, messageID: messageID}
	if _, exists := fb.pending[key]; exists {
		return false
	}

	keys := fb.byChat[chatID]
	if len(keys) >= maxPendingPerChat {
		oldest := keys[0]
		if p, ok := fb.pending[oldest]; ok {
			flushText, flushChatID := p.text, p.chatID
			fb.removeLocked(oldest)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), forwardFlushTimeout)
				defer cancel()
				fb.bot.processIngest(ctx, flushText, flushChatID)
			}()
		}
		keys = keys[1:]
	}

	pf := &pendingForward{text: text, chatID: chatID}
	pf.timer = time.AfterFunc(fb.ttl, func() {
		fb.mu.Lock()
		var flushText string
		var flushChatID int64
		if p, ok := fb.pending[key]; ok {
			flushText, flushChatID = p.text, p.chatID
			fb.removeLocked(key)
		}
		fb.mu.Unlock()
		if flushText != "" {
			ctx, cancel := context.WithTimeout(context.Background(), forwardFlushTimeout)
			defer cancel()
			fb.bot.processIngest(ctx, flushText, flushChatID)
		}
	})
	fb.pending[key] = pf
	fb.byChat[chatID] = append(keys, key)

	return true
}

func (fb *forwardBuffer) remove(chatID int64, messageID int) bool {
	fb.mu.Lock()
	defer fb.mu.Unlock()
	key := bufferKey{chatID: chatID, messageID: messageID}
	if _, ok := fb.pending[key]; ok {
		fb.removeLocked(key)

		return true
	}

	return false
}

func (fb *forwardBuffer) removeLocked(key bufferKey) {
	if pf, ok := fb.pending[key]; ok {
		pf.timer.Stop()
	}
	delete(fb.pending, key)
	keys := fb.byChat[key.chatID]
	for i, k := range keys {
		if k == key {
			fb.byChat[key.chatID] = append(keys[:i], keys[i+1:]...)

			break
		}
	}
}

func (b *Bot) processIngest(ctx context.Context, text string, chatID int64) {
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

type update struct {
	UpdateID int      `json:"update_id"`
	Message  *message `json:"message"`
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

	var chatID int64
	if u.Message.Chat != nil {
		chatID = u.Message.Chat.ID
	}

	// Ветка 1: reply с reply_to_message — объединяем и обрабатываем
	if u.Message.ReplyToMessage != nil {
		b.buffer.remove(chatID, u.Message.ReplyToMessage.MessageID)
		comment := extractText(u.Message)
		forwarded := extractText(u.Message.ReplyToMessage)
		if comment == "" && forwarded == "" {
			clog.FromContext(ctx).Warn("telegram bot: empty reply message ignored")

			return
		}
		text := combineForwardWithComment(comment, forwarded)
		clog.FromContext(ctx).Info("telegram bot: merged forward+reply",
			"chat_id", chatID, "message_id", u.Message.MessageID)
		b.processIngest(ctx, text, chatID)

		return
	}

	// Ветка 2: пересланное без reply
	if len(u.Message.ForwardOrigin) > 0 {
		forwarded := extractText(u.Message)
		if forwarded == "" {
			clog.FromContext(ctx).Warn("telegram bot: empty forwarded message ignored")

			return
		}
		// Если пользователь перед пересылкой написал комментарий — объединяем
		if comment := b.buffer.takeComment(chatID); comment != "" {
			text := combineForwardWithComment(comment, forwarded)
			clog.FromContext(ctx).Info("telegram bot: merged comment+forward",
				"chat_id", chatID, "message_id", u.Message.MessageID)
			b.processIngest(ctx, text, chatID)

			return
		}
		// Иначе — в буфер, ждём возможный reply с комментарием
		if b.buffer.add(ctx, chatID, u.Message.MessageID, forwarded) {
			return
		}
		// дубликат в буфере — обрабатываем пересланное как обычный текст
	}

	// Ветка 3: обычное сообщение — буферизуем, ждём возможную пересылку следом
	text := extractText(u.Message)
	if text == "" {
		clog.FromContext(ctx).Warn("telegram bot: empty message ignored")

		return
	}

	b.buffer.addComment(ctx, chatID, text)
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
