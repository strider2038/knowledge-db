package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/import/session"
	"github.com/strider2038/knowledge-db/internal/import/telegram"
)

const maxImportBodySize = 10 * 1024 * 1024 // 10 MB

// ImportTelegramCreate обрабатывает POST /api/import/telegram.
func (h *Handler) ImportTelegramCreate(w http.ResponseWriter, r *http.Request) {
	if h.sessionStore == nil {
		clog.Warn(r.Context(), "import telegram: not configured")
		writeError(w, http.StatusServiceUnavailable, "import not configured")

		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxImportBodySize))
	if err != nil {
		writeError(w, http.StatusBadRequest, "request body too large or invalid")

		return
	}

	items, err := telegram.ParseChat(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())

		return
	}

	sess, err := h.sessionStore.Create(r.Context(), items)
	if err != nil {
		if errors.Is(err, session.ErrImportNotConfigured) {
			clog.Warn(r.Context(), "import telegram create: not configured")
			writeError(w, http.StatusServiceUnavailable, "import not configured")

			return
		}
		clog.Errorf(r.Context(), "import telegram create: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	clog.Info(r.Context(), "import telegram: session created", "session_id", sess.SessionID, "total", sess.Total)
	current := sess.CurrentItem()
	resp := map[string]any{
		"session_id":    sess.SessionID,
		"total":         sess.Total,
		"current_index": sess.CurrentIndex,
		"current_item":  current,
	}
	writeJSON(w, resp)
}

// ImportTelegramGetSession обрабатывает GET /api/import/telegram/session/:id.
func (h *Handler) ImportTelegramGetSession(w http.ResponseWriter, r *http.Request) {
	if h.sessionStore == nil {
		clog.Warn(r.Context(), "import telegram get session: not configured")
		writeError(w, http.StatusServiceUnavailable, "import not configured")

		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session id required")

		return
	}

	sess, err := h.sessionStore.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, session.ErrImportNotConfigured) {
			clog.Warn(r.Context(), "import telegram get session: not configured")
			writeError(w, http.StatusServiceUnavailable, "import not configured")

			return
		}
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")

			return
		}
		clog.Errorf(r.Context(), "import telegram get session: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	current := sess.CurrentItem()
	resp := map[string]any{
		"session_id":      sess.SessionID,
		"total":           sess.Total,
		"current_index":   sess.CurrentIndex,
		"processed_count": len(sess.ProcessedIDs),
		"rejected_count":  len(sess.RejectedIDs),
		"current_item":    current,
	}
	writeJSON(w, resp)
}

// ImportTelegramAccept обрабатывает POST /api/import/telegram/session/:id/accept.
func (h *Handler) ImportTelegramAccept(w http.ResponseWriter, r *http.Request) {
	if h.sessionStore == nil {
		clog.Warn(r.Context(), "import telegram accept: not configured")
		writeError(w, http.StatusServiceUnavailable, "import not configured")

		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session id required")

		return
	}

	var req struct {
		TypeHint string `json:"type_hint"` //nolint:tagliatelle // REST API snake_case
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	node, nextItem, err := h.sessionStore.Accept(r.Context(), id, req.TypeHint)
	if err != nil {
		if errors.Is(err, session.ErrImportNotConfigured) {
			clog.Warn(r.Context(), "import telegram accept: not configured")
			writeError(w, http.StatusServiceUnavailable, "import not configured")

			return
		}
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")

			return
		}
		if errors.Is(err, session.ErrNoCurrentItem) {
			writeError(w, http.StatusConflict, "no current item")

			return
		}
		clog.Errorf(r.Context(), "import telegram accept: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	clog.Info(r.Context(), "import telegram: accept", "session_id", id, "path", node.Path)
	resp := map[string]any{
		"node":      node,
		"next_item": nextItem,
	}
	writeJSON(w, resp)
}

// ImportTelegramReject обрабатывает POST /api/import/telegram/session/:id/reject.
func (h *Handler) ImportTelegramReject(w http.ResponseWriter, r *http.Request) {
	if h.sessionStore == nil {
		clog.Warn(r.Context(), "import telegram reject: not configured")
		writeError(w, http.StatusServiceUnavailable, "import not configured")

		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "session id required")

		return
	}

	nextItem, err := h.sessionStore.Reject(r.Context(), id)
	if err != nil {
		if errors.Is(err, session.ErrImportNotConfigured) {
			clog.Warn(r.Context(), "import telegram reject: not configured")
			writeError(w, http.StatusServiceUnavailable, "import not configured")

			return
		}
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")

			return
		}
		if errors.Is(err, session.ErrNoCurrentItem) {
			writeError(w, http.StatusConflict, "no current item")

			return
		}
		clog.Errorf(r.Context(), "import telegram reject: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	clog.Info(r.Context(), "import telegram: reject", "session_id", id)
	resp := map[string]any{
		"next_item": nextItem,
	}
	writeJSON(w, resp)
}
