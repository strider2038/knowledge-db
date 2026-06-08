package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	job, err := h.startDumpImagesJob(r.Context(), path)
	if err != nil {
		writeError(w, httpStatusFromJobErr(err), err.Error())

		return
	}
	writeJSON(w, dumpImagesOperationFromJob(job))
}

// GetNodeDumpImagesStatus обрабатывает GET /api/node-dump-images/{id}.
func (h *Handler) GetNodeDumpImagesStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")

		return
	}
	job, ok := h.jobs.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "operation not found")

		return
	}
	writeJSON(w, dumpImagesOperationFromJob(job))
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
	resp, ok := h.jobs.GetLogs(id, after)
	if !ok {
		writeError(w, http.StatusNotFound, "operation not found")

		return
	}
	entries := make([]normalizeLogEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, normalizeLogEntry(e))
	}
	writeJSON(w, dumpImagesLogsResponse{Entries: entries, NextOffset: resp.NextOffset})
}

func (h *Handler) startDumpImagesJob(ctx context.Context, path string) (Job, error) {
	if path == "" {
		return Job{}, errJobPathRequired
	}
	node, err := kb.GetNode(ctx, h.dataPath, path)
	if err != nil {
		return Job{}, err
	}
	nodeType, _ := node.Metadata["type"].(string)
	if nodeType != nodeTypeArticle {
		return Job{}, errJobNodeNotArticle
	}
	if running, ok := h.jobs.FindRunning(jobTypeDumpImages, path); ok {
		return running, errJobDumpAlreadyRunning
	}
	job := h.jobs.Create(jobTypeDumpImages, path, "dump", map[string]any{
		"node_path": path,
		"dump_ok":   false,
		"sync_done": false,
	})
	h.jobs.SetRunning(job.ID, "dump")
	h.jobs.AppendLog(job.ID, "system", "dump images started")
	updated, _ := h.jobs.Get(job.ID)
	go h.runNodeDumpImagesJob(context.WithoutCancel(ctx), job.ID, path)

	return updated, nil
}

func (h *Handler) runNodeDumpImagesJob(ctx context.Context, jobID, nodePath string) {
	themePath, slug := splitArticlePath(nodePath)
	if slug == "" {
		h.jobs.CompleteError(jobID, "dump", "invalid node path", map[string]any{"dump_ok": false, "sync_done": false})
		h.jobs.AppendLog(jobID, "system", "dump images failed: invalid node path")

		return
	}

	fs := afero.NewOsFs()
	client := &http.Client{Timeout: 30 * time.Second}
	modified, downloadErrors, _, err := kb.RunDumpImages(ctx, fs, client, h.dataPath, themePath, slug, false, func(url, targetPath string, size int64) {
		h.jobs.AppendLog(jobID, "stdout", fmt.Sprintf("downloaded %s -> %s (%d bytes)", url, targetPath, size))
	})
	if err != nil {
		h.jobs.CompleteError(jobID, "dump", err.Error(), map[string]any{"dump_ok": false, "sync_done": false})
		h.jobs.AppendLog(jobID, "system", "dump images failed: "+err.Error())

		return
	}
	for _, de := range downloadErrors {
		h.jobs.AppendLog(jobID, "stderr", fmt.Sprintf("%s: %v", de.URL, de.Err))
	}
	if !modified {
		h.jobs.AppendLog(jobID, "system", "no remote images found or no markdown updates required")
	} else {
		h.notifyIndexNodesChanged(ctx, nodePath)
	}

	h.jobs.SetStage(jobID, "sync")
	if h.gitDisabled || h.gitCommitter == nil {
		h.jobs.CompleteSuccess(jobID, "done", map[string]any{"dump_ok": true, "sync_done": false})
		h.jobs.AppendLog(jobID, "system", "dump images completed")

		return
	}

	if err := h.gitCommitter.Sync(ctx); err != nil {
		errText := fmt.Sprintf("sync error: %v", err)
		h.jobs.CompleteError(jobID, "sync", errText, map[string]any{"dump_ok": true, "sync_done": false})
		h.jobs.AppendLog(jobID, "system", "dump images failed: "+errText)

		return
	}

	h.jobs.CompleteSuccess(jobID, "done", map[string]any{"dump_ok": true, "sync_done": true})
	h.jobs.AppendLog(jobID, "system", "dump images completed")
}

func dumpImagesOperationFromJob(job Job) dumpImagesOperation {
	out := dumpImagesOperation{
		ID:         job.ID,
		NodePath:   metadataString(job.Meta, "node_path", job.Target),
		Status:     job.Status,
		Stage:      job.Stage,
		Error:      job.Error,
		StartedAt:  job.StartedAt,
		FinishedAt: job.FinishedAt,
		NextOffset: job.NextOffset,
	}
	out.SyncDone, _ = job.Meta["sync_done"].(bool)
	out.DumpOK, _ = job.Meta["dump_ok"].(bool)

	return out
}
