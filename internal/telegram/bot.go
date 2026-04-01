package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"

	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/strider2038/knowledge-db/internal/pkg/urlutil"
)

const (
	forwardBufferTTL    = 3 * time.Second
	forwardFlushTimeout = 120 * time.Second
	maxPendingPerChat   = 10
)

// messageEntity — сущность разметки Telegram API (MessageEntity).
type messageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"` // для text_link
}

// message — структура сообщения Telegram API (подмножество полей).
type message struct {
	MessageID       int             `json:"message_id"`
	Text            string          `json:"text"`
	Caption         string          `json:"caption"`
	Entities        []messageEntity `json:"entities,omitempty"`
	CaptionEntities []messageEntity `json:"caption_entities,omitempty"`
	ReplyToMessage  *message        `json:"reply_to_message"`
	ForwardOrigin   json.RawMessage `json:"forward_origin,omitempty"`
	Chat            *struct {
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

// extractTextWithFormatting извлекает text или caption с учётом entities, конвертируя в Markdown.
func extractTextWithFormatting(msg *message) string {
	if msg == nil {
		return ""
	}
	var text string
	var entities []messageEntity
	switch {
	case msg.Text != "":
		text = msg.Text
		entities = msg.Entities
	case msg.Caption != "":
		text = msg.Caption
		entities = msg.CaptionEntities
	default:
		return ""
	}

	return entitiesToMarkdown(text, entities)
}

// entitiesToMarkdown конвертирует текст с Telegram entities в Markdown.
// offset и length в entities заданы в UTF-16 code units (Telegram API).
func entitiesToMarkdown(text string, entities []messageEntity) string {
	if len(entities) == 0 {
		return text
	}
	// Сортируем по offset descending, чтобы вставлять разметку с конца и не сбивать позиции.
	sorted := make([]messageEntity, len(entities))
	copy(sorted, entities)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Offset > sorted[j].Offset
	})

	runes := []rune(text)
	utf16ToRune := buildUTF16ToRuneMap(runes)

	for _, e := range sorted {
		startRune, ok := utf16ToRune[e.Offset]
		if !ok || startRune >= len(runes) {
			continue
		}
		endUTF16 := e.Offset + e.Length
		endRune, ok := utf16ToRune[endUTF16]
		if !ok {
			endRune = len(runes)
		}
		if endRune <= startRune {
			continue
		}
		slice := string(runes[startRune:endRune])
		var wrapped string
		switch e.Type {
		case "bold":
			wrapped = "**" + slice + "**"
		case "italic":
			wrapped = "*" + slice + "*"
		case "underline":
			wrapped = "<u>" + slice + "</u>" // Obsidian поддерживает HTML
		case "strikethrough":
			wrapped = "~~" + slice + "~~"
		case "spoiler":
			wrapped = "||" + slice + "||"
		case "blockquote", "expandable_blockquote":
			wrapped = "> " + strings.ReplaceAll(slice, "\n", "\n> ")
		case "code":
			wrapped = "`" + slice + "`"
		case "pre":
			wrapped = "```\n" + slice + "\n```"
		case "text_link":
			if e.URL != "" {
				wrapped = "[" + slice + "](" + e.URL + ")"
			} else {
				wrapped = slice
			}
		case "url":
			wrapped = "[" + slice + "](" + slice + ")"
		default:
			// mention, hashtag, bot_command, email, phone_number, text_mention, custom_emoji — оставляем как есть
			continue
		}
		runes = append(runes[:startRune], append([]rune(wrapped), runes[endRune:]...)...)
	}

	return string(runes)
}

// buildUTF16ToRuneMap строит маппинг: UTF-16 offset -> индекс rune в слайсе.
func buildUTF16ToRuneMap(runes []rune) map[int]int {
	m := make(map[int]int)
	m[0] = 0
	utf16Pos := 0
	for i, r := range runes {
		utf16Pos += utf16Len(r)
		m[utf16Pos] = i + 1
	}

	return m
}

func utf16Len(r rune) int {
	if r <= 0xFFFF {
		return 1
	}

	return 2 // surrogate pair
}

// combineForwardWithComment объединяет комментарий и пересланный контент с явными метками для LLM.
func combineForwardWithComment(comment, forwarded string) string {
	return "Инструкции пользователя: " + comment + "\nПересланное сообщение: " + forwarded
}

// parseForwardOrigin извлекает sourceURL и sourceAuthor из forward_origin Telegram API.
func parseForwardOrigin(raw json.RawMessage) (string, string) {
	var sourceURL, sourceAuthor string
	if len(raw) == 0 {
		return "", ""
	}
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return "", ""
	}
	switch base.Type {
	case "channel":
		var o struct {
			Chat struct {
				ID       int64  `json:"id"`
				Username string `json:"username"`
				Title    string `json:"title"`
			} `json:"chat"`
			MessageID       int    `json:"message_id"`
			AuthorSignature string `json:"author_signature"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}
		if o.Chat.Username != "" {
			sourceURL = fmt.Sprintf("https://t.me/%s/%d", o.Chat.Username, o.MessageID)
		} else if o.Chat.ID < 0 {
			// -100xxxxxxxxxx -> xxxxxxxxxx для t.me/c/xxx/msgId
			channelID := -o.Chat.ID - 1000000000000
			sourceURL = fmt.Sprintf("https://t.me/c/%d/%d", channelID, o.MessageID)
		}
		switch {
		case o.Chat.Title != "":
			sourceAuthor = o.Chat.Title
		case o.Chat.Username != "":
			sourceAuthor = "@" + o.Chat.Username
		case o.AuthorSignature != "":
			sourceAuthor = o.AuthorSignature
		}

		return sourceURL, sourceAuthor
	case "user":
		var o struct {
			SenderUser struct {
				Username  string `json:"username"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
			} `json:"sender_user"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}
		if o.SenderUser.Username != "" {
			sourceAuthor = "@" + o.SenderUser.Username
		} else {
			sourceAuthor = strings.TrimSpace(o.SenderUser.FirstName + " " + o.SenderUser.LastName)
		}

		return "", sourceAuthor
	case "hidden_user":
		var o struct {
			SenderUserName string `json:"sender_user_name"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}

		return "", o.SenderUserName
	case "chat":
		var o struct {
			SenderChat struct {
				Title    string `json:"title"`
				Username string `json:"username"`
			} `json:"sender_chat"`
		}
		if err := json.Unmarshal(raw, &o); err != nil {
			return "", ""
		}
		if o.SenderChat.Title != "" {
			sourceAuthor = o.SenderChat.Title
		} else if o.SenderChat.Username != "" {
			sourceAuthor = "@" + o.SenderChat.Username
		}

		return "", sourceAuthor
	}

	return "", ""
}

type bufferKey struct {
	chatID    int64
	messageID int
}

type pendingForward struct {
	text         string
	chatID       int64
	sourceURL    string
	sourceAuthor string
	timer        *time.Timer
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
		fb.bot.processIngest(ctx, text, chatID, "", "")
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
func (fb *forwardBuffer) add(ctx context.Context, chatID int64, messageID int, text, sourceURL, sourceAuthor string) bool {
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
			flushURL, flushAuthor := p.sourceURL, p.sourceAuthor
			fb.removeLocked(oldest)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), forwardFlushTimeout)
				defer cancel()
				fb.bot.processIngest(ctx, flushText, flushChatID, flushURL, flushAuthor)
			}()
		}
		keys = keys[1:]
	}

	pf := &pendingForward{text: text, chatID: chatID, sourceURL: sourceURL, sourceAuthor: sourceAuthor}
	pf.timer = time.AfterFunc(fb.ttl, func() {
		fb.mu.Lock()
		var flushText string
		var flushChatID int64
		var flushURL, flushAuthor string
		if p, ok := fb.pending[key]; ok {
			flushText, flushChatID = p.text, p.chatID
			flushURL, flushAuthor = p.sourceURL, p.sourceAuthor
			fb.removeLocked(key)
		}
		fb.mu.Unlock()
		if flushText != "" {
			ctx, cancel := context.WithTimeout(context.Background(), forwardFlushTimeout)
			defer cancel()
			fb.bot.processIngest(ctx, flushText, flushChatID, flushURL, flushAuthor)
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

func (b *Bot) processIngest(ctx context.Context, text string, chatID int64, sourceURL, sourceAuthor string) {
	clog.Info(ctx, "telegram bot: process ingest",
		"chat_id", chatID,
		"text_len", len(text),
		"has_http", strings.Contains(text, "http://") || strings.Contains(text, "https://"),
		"source_url", sourceURL,
		"source_author", sourceAuthor)

	if sourceURL != "" {
		sourceURL = urlutil.StripTrackingParamsFromURL(sourceURL)
	}
	if chatID != 0 {
		_ = b.sendMessage(ctx, chatID, "Принял, обрабатываю...")
	}
	node, err := b.ingester.IngestText(ctx, ingestion.IngestRequest{
		Text:         text,
		SourceURL:    sourceURL,
		SourceAuthor: sourceAuthor,
	})
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
		clog.Warn(ctx, "telegram bot: unauthorized message ignored",
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
		comment := extractTextWithFormatting(u.Message)
		forwarded := extractTextWithFormatting(u.Message.ReplyToMessage)
		if comment == "" && forwarded == "" {
			clog.Warn(ctx, "telegram bot: empty reply message ignored")

			return
		}
		text := combineForwardWithComment(comment, forwarded)
		sourceURL, sourceAuthor := parseForwardOrigin(u.Message.ReplyToMessage.ForwardOrigin)
		clog.Info(ctx, "telegram bot: merged forward+reply",
			"chat_id", chatID, "message_id", u.Message.MessageID)
		b.processIngest(ctx, text, chatID, sourceURL, sourceAuthor)

		return
	}

	// Ветка 2: пересланное без reply
	if len(u.Message.ForwardOrigin) > 0 {
		forwarded := extractTextWithFormatting(u.Message)
		if forwarded == "" {
			clog.Warn(ctx, "telegram bot: empty forwarded message ignored")

			return
		}
		sourceURL, sourceAuthor := parseForwardOrigin(u.Message.ForwardOrigin)
		// Если пользователь перед пересылкой написал комментарий — объединяем
		if comment := b.buffer.takeComment(chatID); comment != "" {
			text := combineForwardWithComment(comment, forwarded)
			clog.Info(ctx, "telegram bot: merged comment+forward",
				"chat_id", chatID, "message_id", u.Message.MessageID)
			b.processIngest(ctx, text, chatID, sourceURL, sourceAuthor)

			return
		}
		// Иначе — в буфер, ждём возможный reply с комментарием
		if b.buffer.add(ctx, chatID, u.Message.MessageID, forwarded, sourceURL, sourceAuthor) {
			return
		}
		// дубликат в буфере — обрабатываем пересланное как обычный текст
		b.processIngest(ctx, forwarded, chatID, sourceURL, sourceAuthor)

		return
	}

	// Ветка 3: обычное сообщение — буферизуем, ждём возможную пересылку следом
	text := extractTextWithFormatting(u.Message)
	if text == "" {
		clog.Warn(ctx, "telegram bot: empty message ignored")

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
