package translationqueue

import (
	"sync"
	"time"
)

// Status — статус задачи перевода.
const (
	StatusNone       = "none"
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusFailed     = "failed"
)

// TranslationStatus — состояние задачи перевода.
type TranslationStatus struct {
	Status       string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

// ArticleKey — ключ статьи (themePath/slug).
type ArticleKey struct {
	ThemePath string
	Slug      string
}

func (k ArticleKey) String() string {
	if k.ThemePath == "" {
		return k.Slug
	}

	return k.ThemePath + "/" + k.Slug
}

// Queue — in-memory очередь переводов с отслеживанием статусов.
type Queue struct {
	mu      sync.RWMutex
	status  map[string]*TranslationStatus
	channel chan ArticleKey
}

// New создаёт Queue с заданной ёмкостью канала.
func New(channelCapacity int) *Queue {
	if channelCapacity <= 0 {
		channelCapacity = 100
	}

	return &Queue{
		status:  make(map[string]*TranslationStatus),
		channel: make(chan ArticleKey, channelCapacity),
	}
}

// Channel возвращает канал для чтения задач воркером.
func (q *Queue) Channel() <-chan ArticleKey {
	return q.channel
}

// Enqueue ставит статью в очередь. Если статус уже pending или in_progress,
// не создаёт дубликат и возвращает текущий статус.
// Возвращает (status, alreadyQueued).
func (q *Queue) Enqueue(themePath, slug string) (string, bool) {
	key := ArticleKey{ThemePath: themePath, Slug: slug}
	keyStr := key.String()

	q.mu.Lock()
	defer q.mu.Unlock()

	if s, ok := q.status[keyStr]; ok {
		if s.Status == StatusPending || s.Status == StatusInProgress {
			return s.Status, true
		}
		// done/failed — можно поставить заново (повторная попытка)
	}

	q.status[keyStr] = &TranslationStatus{
		Status: StatusPending,
	}

	q.channel <- key

	return StatusPending, false
}

// GetStatus возвращает статус по ключу статьи. Если записи нет — "none".
func (q *Queue) GetStatus(themePath, slug string) (string, string) {
	keyStr := ArticleKey{ThemePath: themePath, Slug: slug}.String()

	q.mu.RLock()
	defer q.mu.RUnlock()

	if s, ok := q.status[keyStr]; ok {
		return s.Status, s.ErrorMessage
	}

	return StatusNone, ""
}

// SetInProgress переводит статус в in_progress.
func (q *Queue) SetInProgress(themePath, slug string) {
	keyStr := ArticleKey{ThemePath: themePath, Slug: slug}.String()
	q.mu.Lock()
	defer q.mu.Unlock()
	if s, ok := q.status[keyStr]; ok {
		s.Status = StatusInProgress
		s.StartedAt = time.Now()
	}
}

// SetDone переводит статус в done.
func (q *Queue) SetDone(themePath, slug string) {
	keyStr := ArticleKey{ThemePath: themePath, Slug: slug}.String()
	q.mu.Lock()
	defer q.mu.Unlock()
	if s, ok := q.status[keyStr]; ok {
		s.Status = StatusDone
		s.FinishedAt = time.Now()
		s.ErrorMessage = ""
	}
}

// SetFailed переводит статус в failed и сохраняет сообщение об ошибке.
func (q *Queue) SetFailed(themePath, slug string, errMsg string) {
	keyStr := ArticleKey{ThemePath: themePath, Slug: slug}.String()
	q.mu.Lock()
	defer q.mu.Unlock()
	if s, ok := q.status[keyStr]; ok {
		s.Status = StatusFailed
		s.FinishedAt = time.Now()
		s.ErrorMessage = errMsg
	}
}
