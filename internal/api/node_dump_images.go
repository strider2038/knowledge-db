package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/spf13/afero"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type dumpImagesOperation struct {
	ID         string              `json:"id"`
	NodePath   string              `json:"node_path"`
	Status     string              `json:"status"`
	Stage      string              `json:"stage"`
	Error      string              `json:"error,omitempty"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt *time.Time          `json:"finished_at,omitempty"`
	SyncDone   bool                `json:"sync_done"`
	DumpOK     bool                `json:"dump_ok"`
	Logs       []normalizeLogEntry `json:"-"`
	NextOffset int64               `json:"-"`
}

type dumpImagesLogsResponse struct {
	Entries    []normalizeLogEntry `json:"entries"`
	NextOffset int64               `json:"next_offset"`
}

// PostNodeDumpImages обрабатывает POST /api/nodes/{path...}/dump-images.
func (h *Handler) PostNodeDumpImages(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.PathValue("path"), "/dump-images")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}

	node, err := kb.GetNode(r.Context(), h.dataPath, path)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	nodeType, _ := node.Metadata["type"].(string)
	if nodeType != "article" {
		writeError(w, http.StatusBadRequest, "node is not an article")

		return
	}

	h.dumpImagesMu.RLock()
	for _, op := range h.dumpImagesOps {
		if op.NodePath == path && op.Status == "running" {
			h.dumpImagesMu.RUnlock()
			writeError(w, http.StatusConflict, "dump images already running for this node")

			return
		}
	}
	h.dumpImagesMu.RUnlock()

	op := dumpImagesOperation{
		ID:        uuid.NewString(),
		NodePath:  path,
		Status:    "running",
		Stage:     "dump",
		StartedAt: time.Now().UTC(),
	}
	h.dumpImagesMu.Lock()
	h.appendDumpImagesLogLocked(&op, "system", "dump images started")
	h.dumpImagesOps[op.ID] = op
	h.dumpImagesMu.Unlock()

	go h.runNodeDumpImages(context.WithoutCancel(r.Context()), op)

	writeJSON(w, op)
}

// GetNodeDumpImagesStatus обрабатывает GET /api/node-dump-images/{id}.
func (h *Handler) GetNodeDumpImagesStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")

		return
	}
	h.dumpImagesMu.RLock()
	op, ok := h.dumpImagesOps[id]
	h.dumpImagesMu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "operation not found")

		return
	}
	writeJSON(w, op)
}

// GetNodeDumpImagesLogs обрабатывает GET /api/node-dump-images/{id}/logs.
func (h *Handler) GetNodeDumpImagesLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")

		return
	}
	after := int64(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid after")

			return
		}
		after = v
	}
	h.dumpImagesMu.RLock()
	op, ok := h.dumpImagesOps[id]
	h.dumpImagesMu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "operation not found")

		return
	}
	entries := make([]normalizeLogEntry, 0)
	for _, e := range op.Logs {
		if e.Offset > after {
			entries = append(entries, e)
		}
	}
	writeJSON(w, dumpImagesLogsResponse{Entries: entries, NextOffset: op.NextOffset})
}

func (h *Handler) runNodeDumpImages(ctx context.Context, op dumpImagesOperation) {
	themePath, slug := splitArticlePath(op.NodePath)
	if slug == "" {
		h.completeDumpImagesOp(ctx, op.ID, "error", "dump", "invalid node path", false, false)

		return
	}

	fs := afero.NewOsFs()
	client := &http.Client{Timeout: 30 * time.Second}
	modified, downloadErrors, _, err := kb.RunDumpImages(ctx, fs, client, h.dataPath, themePath, slug, false, func(url, targetPath string, size int64) {
		h.appendDumpImagesLog(op.ID, "stdout", fmt.Sprintf("downloaded %s -> %s (%d bytes)", url, targetPath, size))
	})
	if err != nil {
		h.completeDumpImagesOp(ctx, op.ID, "error", "dump", err.Error(), false, false)

		return
	}
	for _, de := range downloadErrors {
		h.appendDumpImagesLog(op.ID, "stderr", fmt.Sprintf("%s: %v", de.URL, de.Err))
	}
	if !modified {
		h.appendDumpImagesLog(op.ID, "system", "no remote images found or no markdown updates required")
	}

	h.updateDumpImagesOpStage(op.ID, "sync")
	if h.gitDisabled || h.gitCommitter == nil {
		h.completeDumpImagesOp(ctx, op.ID, "success", "done", "", true, false)

		return
	}

	if err := h.gitCommitter.Sync(ctx); err != nil {
		h.completeDumpImagesOp(ctx, op.ID, "error", "sync", fmt.Sprintf("sync error: %v", err), true, false)

		return
	}

	h.completeDumpImagesOp(ctx, op.ID, "success", "done", "", true, true)
}

func (h *Handler) appendDumpImagesLog(id, stream, text string) {
	h.dumpImagesMu.Lock()
	defer h.dumpImagesMu.Unlock()
	op, ok := h.dumpImagesOps[id]
	if !ok {
		return
	}
	h.appendDumpImagesLogLocked(&op, stream, text)
	h.dumpImagesOps[id] = op
}

func (h *Handler) appendDumpImagesLogLocked(op *dumpImagesOperation, stream, text string) {
	op.NextOffset++
	op.Logs = append(op.Logs, normalizeLogEntry{
		Offset:    op.NextOffset,
		Stream:    stream,
		Text:      text,
		Timestamp: time.Now().UTC(),
	})
	if len(op.Logs) > 1000 {
		op.Logs = append([]normalizeLogEntry(nil), op.Logs[len(op.Logs)-1000:]...)
	}
}

func (h *Handler) updateDumpImagesOpStage(id, stage string) {
	h.dumpImagesMu.Lock()
	defer h.dumpImagesMu.Unlock()
	op, ok := h.dumpImagesOps[id]
	if !ok {
		return
	}
	op.Stage = stage
	h.dumpImagesOps[id] = op
}

func (h *Handler) completeDumpImagesOp(ctx context.Context, id, status, stage, errText string, dumpOK, syncDone bool) {
	h.dumpImagesMu.Lock()
	defer h.dumpImagesMu.Unlock()
	op, ok := h.dumpImagesOps[id]
	if !ok {
		return
	}
	op.Status = status
	op.Stage = stage
	op.Error = errText
	op.DumpOK = dumpOK
	op.SyncDone = syncDone
	finishedAt := time.Now().UTC()
	op.FinishedAt = &finishedAt
	if status == "success" {
		h.appendDumpImagesLogLocked(&op, "system", "dump images completed")
	} else {
		h.appendDumpImagesLogLocked(&op, "system", "dump images failed: "+errText)
	}
	h.dumpImagesOps[id] = op
	if status == "error" {
		clog.Error(ctx, "node dump images failed", "node_path", op.NodePath, "stage", stage, "error", errText)
	}
}
