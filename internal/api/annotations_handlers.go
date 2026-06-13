package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// ListNodeAnnotations handles GET /api/nodes/{path...}/annotations.
func (h *Handler) ListNodeAnnotations(w http.ResponseWriter, r *http.Request) {
	nodePath := strings.TrimSpace(r.PathValue("path"))
	if nodePath == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	notes, err := kb.ListNodeAnnotations(r.Context(), h.dataPath, nodePath)
	if err != nil {
		writeAnnotationError(w, r, err)

		return
	}
	writeJSON(w, map[string]any{"notes": notes})
}

// CreateNodeAnnotation handles POST /api/nodes/{path...}/annotations.
func (h *Handler) CreateNodeAnnotation(w http.ResponseWriter, r *http.Request) {
	nodePath := strings.TrimSpace(r.PathValue("path"))
	if nodePath == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	var req struct {
		Body   string               `json:"body"`
		Anchor *kb.AnnotationAnchor `json:"anchor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	note, err := kb.CreateNodeAnnotation(r.Context(), h.dataPath, nodePath, kb.CreateAnnotationParams{
		Body:   req.Body,
		Anchor: req.Anchor,
	})
	if err != nil {
		writeAnnotationError(w, r, err)

		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, note)
}

// UpdateNodeAnnotation handles PATCH /api/nodes/{path...}/annotations/{id}.
func (h *Handler) UpdateNodeAnnotation(w http.ResponseWriter, r *http.Request) {
	nodePath := strings.TrimSpace(r.PathValue("path"))
	noteID := strings.TrimSpace(r.PathValue("id"))
	if noteID == "" {
		var ok bool
		nodePath, noteID, ok = annotationsItemPath(nodePath)
		if !ok {
			writeError(w, http.StatusBadRequest, "path required")

			return
		}
	}
	if nodePath == "" || noteID == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	var req struct {
		Body   *string              `json:"body"`
		Anchor *kb.AnnotationAnchor `json:"anchor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if req.Body == nil && req.Anchor == nil {
		writeError(w, http.StatusBadRequest, "body or anchor required")

		return
	}
	note, err := kb.UpdateNodeAnnotation(r.Context(), h.dataPath, nodePath, noteID, kb.UpdateAnnotationParams{
		Body:   req.Body,
		Anchor: req.Anchor,
	})
	if err != nil {
		writeAnnotationError(w, r, err)

		return
	}
	writeJSON(w, note)
}

// DeleteNodeAnnotation handles DELETE /api/nodes/{path...}/annotations/{id}.
func (h *Handler) DeleteNodeAnnotation(w http.ResponseWriter, r *http.Request) {
	nodePath := strings.TrimSpace(r.PathValue("path"))
	noteID := strings.TrimSpace(r.PathValue("id"))
	if noteID == "" {
		var ok bool
		nodePath, noteID, ok = annotationsItemPath(nodePath)
		if !ok {
			writeError(w, http.StatusBadRequest, "path required")

			return
		}
	}
	if nodePath == "" || noteID == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	if err := kb.DeleteNodeAnnotation(r.Context(), h.dataPath, nodePath, noteID); err != nil {
		writeAnnotationError(w, r, err)

		return
	}
	writeJSON(w, map[string]any{"id": noteID, "deleted": true})
}

func writeAnnotationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, kb.ErrNodeNotFound), errors.Is(err, kb.ErrAnnotationNotFound):
		clog.Debug(r.Context(), "annotations: not found", "error", err)
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, kb.ErrInvalidAnnotation):
		clog.Debug(r.Context(), "annotations: invalid", "error", err)
		writeError(w, http.StatusBadRequest, "invalid annotation")
	default:
		clog.Errorf(r.Context(), "annotations: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
