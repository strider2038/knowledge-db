package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/muonsoft/clog"
)

type createChatSessionRequest struct {
	Title string `json:"title"`
}

type renameChatSessionRequest struct {
	Title string `json:"title"`
}

func (h *Handler) GetChats(w http.ResponseWriter, r *http.Request) {
	if h.chatStore == nil {
		writeError(w, http.StatusServiceUnavailable, "chat store unavailable")
		return
	}
	_ = h.chatStore.CleanupExpired(r.Context())
	sessions, err := h.chatStore.ListSessions(r.Context())
	if err != nil {
		clog.Errorf(r.Context(), "list chats: %w", err)
		writeError(w, http.StatusInternalServerError, "failed to list chats")
		return
	}
	writeJSON(w, map[string]any{"sessions": sessions})
}

func (h *Handler) PostChats(w http.ResponseWriter, r *http.Request) {
	if h.chatStore == nil {
		writeError(w, http.StatusServiceUnavailable, "chat store unavailable")
		return
	}
	var req createChatSessionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	session, err := h.chatStore.CreateSession(r.Context(), uuid.NewString(), strings.TrimSpace(req.Title))
	if err != nil {
		clog.Errorf(r.Context(), "create chat: %w", err)
		writeError(w, http.StatusInternalServerError, "failed to create chat")
		return
	}
	writeJSON(w, session)
}

func (h *Handler) GetChatByID(w http.ResponseWriter, r *http.Request) {
	if h.chatStore == nil {
		writeError(w, http.StatusServiceUnavailable, "chat store unavailable")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "chat id required")
		return
	}
	out, err := h.chatStore.GetSession(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "chat not found")
			return
		}
		clog.Errorf(r.Context(), "get chat: %w", err)
		writeError(w, http.StatusInternalServerError, "failed to load chat")
		return
	}
	writeJSON(w, out)
}

func (h *Handler) PatchChatByID(w http.ResponseWriter, r *http.Request) {
	if h.chatStore == nil {
		writeError(w, http.StatusServiceUnavailable, "chat store unavailable")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "chat id required")
		return
	}
	var req renameChatSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if err := h.chatStore.RenameSession(r.Context(), id, req.Title); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "chat not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to rename chat")
		return
	}
	writeJSON(w, map[string]any{"id": id, "title": strings.TrimSpace(req.Title)})
}

func (h *Handler) DeleteChatByID(w http.ResponseWriter, r *http.Request) {
	if h.chatStore == nil {
		writeError(w, http.StatusServiceUnavailable, "chat store unavailable")
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "chat id required")
		return
	}
	if err := h.chatStore.DeleteSession(r.Context(), id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "chat not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete chat")
		return
	}
	writeJSON(w, map[string]any{"id": id, "deleted": true})
}
