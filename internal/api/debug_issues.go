package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/muonsoft/clog"

	"github.com/strider2038/knowledge-db/internal/debugdata"
)

type debugIssueStore interface {
	WriteIssue(ctx context.Context, payload debugdata.IssuePayload) (debugdata.Issue, error)
	UpdateIssueStatus(ctx context.Context, issueID, status string) (debugdata.Issue, error)
}

func (h *Handler) PostDebugIssue(w http.ResponseWriter, r *http.Request) {
	if h.debugStore == nil {
		writeError(w, http.StatusServiceUnavailable, "debug issue store not configured")

		return
	}
	var req struct {
		Title       string         `json:"title"`
		Description string         `json:"description"`
		Page        string         `json:"page"`
		Context     map[string]any `json:"context"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")

		return
	}
	if strings.TrimSpace(req.Description) == "" {
		writeError(w, http.StatusBadRequest, "description is required")

		return
	}
	issue, err := h.debugStore.WriteIssue(r.Context(), debugdata.IssuePayload{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		Page:        strings.TrimSpace(req.Page),
		Context:     req.Context,
	})
	if err != nil {
		clog.Errorf(r.Context(), "write debug issue: %w", err)
		writeError(w, http.StatusInternalServerError, "failed to save issue")

		return
	}
	writeJSON(w, map[string]any{
		"id":         issue.ID,
		"status":     issue.Status,
		"created_at": issue.CreatedAt,
		"updated_at": issue.UpdatedAt,
	})
}

func (h *Handler) UpdateDebugIssueStatus(w http.ResponseWriter, r *http.Request) {
	if h.debugStore == nil {
		writeError(w, http.StatusServiceUnavailable, "debug issue store not configured")

		return
	}
	var req struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	issueID := strings.TrimSpace(req.ID)
	if issueID == "" {
		writeError(w, http.StatusBadRequest, "id is required")

		return
	}
	status := strings.TrimSpace(req.Status)
	if status != debugdata.IssueStatusNew && status != debugdata.IssueStatusInvestigating && status != debugdata.IssueStatusFixed {
		writeError(w, http.StatusBadRequest, "status must be one of: new, investigating, fixed")

		return
	}
	issue, err := h.debugStore.UpdateIssueStatus(r.Context(), issueID, status)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "issue not found")

			return
		}
		clog.Errorf(r.Context(), "update debug issue status: %w", err)
		writeError(w, http.StatusInternalServerError, "failed to update issue status")

		return
	}
	writeJSON(w, map[string]any{
		"id":         issue.ID,
		"status":     issue.Status,
		"created_at": issue.CreatedAt,
		"updated_at": issue.UpdatedAt,
	})
}
