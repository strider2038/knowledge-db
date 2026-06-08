package api

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/strider2038/knowledge-db/internal/kb"
)

type normalizeOperation struct {
	ID          string              `json:"id"`
	NodePath    string              `json:"node_path"`
	Status      string              `json:"status"`
	Stage       string              `json:"stage"`
	Error       string              `json:"error,omitempty"`
	StartedAt   time.Time           `json:"started_at"`
	FinishedAt  *time.Time          `json:"finished_at,omitempty"`
	SyncDone    bool                `json:"sync_done"`
	NormalizeOK bool                `json:"normalize_ok"`
	Logs        []normalizeLogEntry `json:"-"`
	NextOffset  int64               `json:"-"`
}

type normalizeLogEntry struct {
	Offset    int64     `json:"offset"`
	Stream    string    `json:"stream"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type normalizeLogsResponse struct {
	Entries    []normalizeLogEntry `json:"entries"`
	NextOffset int64               `json:"next_offset"`
}

type nodeNormalizer interface {
	NormalizeNode(ctx context.Context, dataPath string, node *kb.Node, onLog func(stream, text string)) error
}

type cursorNodeNormalizer struct{}

func NewCursorNodeNormalizer() nodeNormalizer {
	return &cursorNodeNormalizer{}
}

func (n *cursorNodeNormalizer) NormalizeNode(ctx context.Context, dataPath string, node *kb.Node, onLog func(stream, text string)) error {
	return RunCursorAgent(ctx, dataPath, buildNormalizationPrompt(node), onLog)
}

func buildNormalizationPrompt(node *kb.Node) string {
	var b strings.Builder
	b.WriteString("Нормализуй текущий markdown-узел базы знаний и сохрани изменения в этом же файле.\\n")
	b.WriteString("Сохрани фактический смысл, исправь форматирование markdown и frontmatter, не удаляй важные данные.\\n")
	b.WriteString("Не изменяй путь узла и не переименовывай файл.\\n")
	b.WriteString("Node path: ")
	b.WriteString(node.Path)
	b.WriteString("\\n\\n")
	appendNodeContext(&b, node)

	return b.String()
}

// PostNodeNormalize обрабатывает POST /api/nodes/{path...}/normalize.
func (h *Handler) PostNodeNormalize(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.PathValue("path"), "/normalize")
	job, err := h.startNormalizeJob(r.Context(), path)
	if err != nil {
		writeError(w, httpStatusFromJobErr(err), err.Error())

		return
	}
	writeJSON(w, normalizeOperationFromJob(job))
}

// GetNodeNormalizeStatus обрабатывает GET /api/node-normalization/{id}.
func (h *Handler) GetNodeNormalizeStatus(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, normalizeOperationFromJob(job))
}

// GetNodeNormalizeLogs обрабатывает GET /api/node-normalization/{id}/logs.
func (h *Handler) GetNodeNormalizeLogs(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, normalizeLogsResponse{Entries: entries, NextOffset: resp.NextOffset})
}

func (h *Handler) startNormalizeJob(ctx context.Context, path string) (Job, error) {
	if path == "" {
		return Job{}, errJobPathRequired
	}
	if h.normalizeRunner == nil {
		return Job{}, errJobNormalizeUnavailable
	}
	if _, ok := h.normalizeRunner.(*cursorNodeNormalizer); ok {
		if _, err := exec.LookPath("cursor-agent"); err != nil {
			return Job{}, errJobCursorAgentUnavailable
		}
	}
	if running, ok := h.jobs.FindRunning(jobTypeNormalize, path); ok {
		return running, errJobNormalizeAlreadyRunning
	}
	node, err := kb.GetNode(ctx, h.dataPath, path)
	if err != nil {
		return Job{}, err
	}

	job := h.jobs.Create(jobTypeNormalize, path, "normalize", map[string]any{
		"node_path":    path,
		"normalize_ok": false,
		"sync_done":    false,
	})
	h.jobs.SetRunning(job.ID, "normalize")
	h.jobs.AppendLog(job.ID, "system", "normalization started")
	updated, _ := h.jobs.Get(job.ID)
	go h.runNodeNormalizationJob(context.WithoutCancel(ctx), job.ID, node)

	return updated, nil
}

func (h *Handler) runNodeNormalizationJob(ctx context.Context, jobID string, node *kb.Node) {
	if err := h.normalizeRunner.NormalizeNode(ctx, h.dataPath, node, func(stream, text string) {
		h.jobs.AppendLog(jobID, stream, text)
	}); err != nil {
		h.jobs.CompleteError(jobID, "normalize", err.Error(), map[string]any{
			"normalize_ok": false,
			"sync_done":    false,
		})
		h.jobs.AppendLog(jobID, "system", "normalization failed: "+err.Error())

		return
	}

	h.notifyIndexNodesChanged(node.Path)
	h.jobs.SetStage(jobID, "sync")
	if h.gitDisabled || h.gitCommitter == nil {
		h.jobs.CompleteSuccess(jobID, "done", map[string]any{
			"normalize_ok": true,
			"sync_done":    false,
		})
		h.jobs.AppendLog(jobID, "system", "normalization completed")

		return
	}

	if err := h.gitCommitter.Sync(ctx); err != nil {
		errText := fmt.Sprintf("sync error: %v", err)
		h.jobs.CompleteError(jobID, "sync", errText, map[string]any{
			"normalize_ok": true,
			"sync_done":    false,
		})
		h.jobs.AppendLog(jobID, "system", "normalization failed: "+errText)

		return
	}

	h.jobs.CompleteSuccess(jobID, "done", map[string]any{
		"normalize_ok": true,
		"sync_done":    true,
	})
	h.jobs.AppendLog(jobID, "system", "normalization completed")
}

func normalizeOperationFromJob(job Job) normalizeOperation {
	out := normalizeOperation{
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
	out.NormalizeOK, _ = job.Meta["normalize_ok"].(bool)

	return out
}

func metadataString(meta map[string]any, key, fallback string) string {
	if meta == nil {
		return fallback
	}
	if v, ok := meta[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}

	return fallback
}

var _ nodeNormalizer = (*cursorNodeNormalizer)(nil)
