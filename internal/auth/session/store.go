package session

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// CookieName — имя cookie сессии.
const CookieName = "kb_session"

// Session — сессия пользователя.
type Session struct {
	ID        string
	ExpiresAt time.Time
}

// Store — потокобезопасное in-memory хранилище сессий.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewStore создаёт новый Store.
func NewStore() *Store {
	s := &Store{sessions: make(map[string]*Session)}
	go s.cleanupLoop()

	return s
}

// Create создаёт новую сессию с заданным TTL. Возвращает ID сессии.
func (s *Store) Create(ttl time.Duration) (string, error) {
	id, err := generateSessionID()
	if err != nil {
		return "", err
	}

	sess := &Session{
		ID:        id,
		ExpiresAt: time.Now().Add(ttl),
	}

	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()

	return id, nil
}

// Get проверяет наличие и валидность сессии. Возвращает true, если сессия валидна.
func (s *Store) Get(id string) bool {
	if id == "" {
		return false
	}

	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()

	if !ok || sess == nil {
		return false
	}

	if time.Now().After(sess.ExpiresAt) {
		s.mu.Lock()
		delete(s.sessions, id)
		s.mu.Unlock()

		return false
	}

	return true
}

// Invalidate удаляет сессию.
func (s *Store) Invalidate(id string) {
	if id == "" {
		return
	}

	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

func (s *Store) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupExpired()
	}
}

func (s *Store) cleanupExpired() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, sess := range s.sessions {
		if now.After(sess.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
