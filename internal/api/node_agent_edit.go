package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/strider2038/knowledge-db/internal/kb"
)

const maxAgentEditInstructionLen = 16 * 1024

type agentEditOperation struct {
	ID         string              `json:"id"`
	NodePath   string              `json:"node_path"`
	Status     string              `json:"status"`
	Stage      string              `json:"stage"`
	Error      string              `json:"error,omitempty"`
	StartedAt  time.Time           `json:"started_at"`
	FinishedAt *time.Time          `json:"finished_at,omitempty"`
	SyncDone   bool                `json:"sync_done"`
	EditOK     bool                `json:"edit_ok"`
	Logs       []agentEditLogEntry `json:"-"`
	NextOffset int64               `json:"-"`
}

type agentEditLogEntry struct {
	Offset    int64     `json:"offset"`
	Stream    string    `json:"stream"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type agentEditLogsResponse struct {
	Entries    []agentEditLogEntry `json:"entries"`
	NextOffset int64               `json:"next_offset"`
}

// NodeAgentEditor редактирует узел через Cursor Agent (для тестов и bootstrap).
type NodeAgentEditor interface {
	EditNode(ctx context.Context, dataPath string, node *kb.Node, instruction string, onLog func(stream, text string)) error
}

type cursorNodeAgentEditor struct{}

func NewCursorNodeAgentEditor() NodeAgentEditor {
	return &cursorNodeAgentEditor{}
}

func (e *cursorNodeAgentEditor) EditNode(ctx context.Context, dataPath string, node *kb.Node, instruction string, onLog func(stream, text string)) error {
	return RunCursorAgent(ctx, dataPath, buildAgentEditPrompt(node, instruction), onLog)
}

func buildAgentEditPrompt(node *kb.Node, instruction string) string {
	var b strings.Builder
	b.WriteString("Отредактируй markdown-узел базы знаний по инструкции пользователя и сохрани изменения в этом же файле.\n")
	b.WriteString("Редактируй ТОЛЬКО файл узла по указанному path. Не изменяй путь узла и не переименовывай файл.\n")
	b.WriteString("Сохрани фактический смысл, если инструкция не требует иного.\n")
	b.WriteString("Node path: ")
	b.WriteString(node.Path)
	b.WriteString("\n\n")
	b.WriteString("Инструкция пользователя:\n")
	b.WriteString(instruction)
	b.WriteString("\n\n")
	appendNodeContext(&b, node)

	return b.String()
}

func appendNodeContext(b *strings.Builder, node *kb.Node) {
	b.WriteString("Frontmatter and content:\\n")
	metaJSON, err := json.MarshalIndent(node.Metadata, "", "  ")
	if err != nil {
		metaJSON = []byte("{}")
	}
	b.WriteString("metadata: ")
	b.Write(metaJSON)
	b.WriteString("\\n\\nannotation:\\n")
	b.WriteString(node.Annotation)
	b.WriteString("\\n\\ncontent:\\n")
	b.WriteString(node.Content)
}

// PostNodeAgentEdit обрабатывает POST /api/nodes/{path...}/agent-edit.
func (h *Handler) PostNodeAgentEdit(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.PathValue("path"), "/agent-edit")
	var req struct {
		Instruction string `json:"instruction"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	job, err := h.startAgentEditJob(r.Context(), path, req.Instruction)
	if err != nil {
		writeError(w, httpStatusFromJobErr(err), err.Error())

		return
	}
	writeJSON(w, agentEditOperationFromJob(job))
}

// GetNodeAgentEditStatus обрабатывает GET /api/node-agent-edit/{id}.
func (h *Handler) GetNodeAgentEditStatus(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, agentEditOperationFromJob(job))
}

// GetNodeAgentEditLogs обрабатывает GET /api/node-agent-edit/{id}/logs.
func (h *Handler) GetNodeAgentEditLogs(w http.ResponseWriter, r *http.Request) {
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
	entries := make([]agentEditLogEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, agentEditLogEntry(e))
	}
	writeJSON(w, agentEditLogsResponse{Entries: entries, NextOffset: resp.NextOffset})
}

func (h *Handler) startAgentEditJob(ctx context.Context, path, instruction string) (Job, error) {
	if path == "" {
		return Job{}, errJobPathRequired
	}
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return Job{}, errJobAgentEditInstructionRequired
	}
	if len(instruction) > maxAgentEditInstructionLen {
		return Job{}, errJobAgentEditInstructionTooLong
	}
	if h.agentEditRunner == nil {
		return Job{}, errJobAgentEditUnavailable
	}
	if _, ok := h.agentEditRunner.(*cursorNodeAgentEditor); ok {
		if _, err := exec.LookPath("cursor-agent"); err != nil {
			return Job{}, errJobCursorAgentUnavailable
		}
	}
	if running, ok := h.jobs.FindRunning(jobTypeAgentEdit, path); ok {
		return running, errJobAgentEditAlreadyRunning
	}
	node, err := kb.GetNode(ctx, h.dataPath, path)
	if err != nil {
		return Job{}, err
	}

	job := h.jobs.Create(jobTypeAgentEdit, path, "edit", map[string]any{
		"node_path": path,
		"edit_ok":   false,
		"sync_done": false,
	})
	h.jobs.SetRunning(job.ID, "edit")
	h.jobs.AppendLog(job.ID, "system", "agent edit started")
	updated, _ := h.jobs.Get(job.ID)
	go h.runNodeAgentEditJob(context.WithoutCancel(ctx), job.ID, node, instruction)

	return updated, nil
}

func (h *Handler) runNodeAgentEditJob(ctx context.Context, jobID string, node *kb.Node, instruction string) {
	if err := h.agentEditRunner.EditNode(ctx, h.dataPath, node, instruction, func(stream, text string) {
		h.jobs.AppendLog(jobID, stream, text)
	}); err != nil {
		h.jobs.CompleteError(jobID, "edit", err.Error(), map[string]any{
			"edit_ok":   false,
			"sync_done": false,
		})
		h.jobs.AppendLog(jobID, "system", "agent edit failed: "+err.Error())

		return
	}

	h.notifyIndexNodesChanged(node.Path)
	h.jobs.SetStage(jobID, "sync")
	h.jobs.AppendLog(jobID, "system", "stage: sync")
	if h.gitDisabled || h.gitCommitter == nil {
		h.jobs.CompleteSuccess(jobID, "done", map[string]any{
			"edit_ok":   true,
			"sync_done": false,
		})
		h.jobs.AppendLog(jobID, "system", "agent edit completed")

		return
	}

	if err := h.gitCommitter.Sync(ctx); err != nil {
		errText := fmt.Sprintf("sync error: %v", err)
		h.jobs.CompleteError(jobID, "sync", errText, map[string]any{
			"edit_ok":   true,
			"sync_done": false,
		})
		h.jobs.AppendLog(jobID, "system", "agent edit failed: "+errText)

		return
	}

	h.jobs.CompleteSuccess(jobID, "done", map[string]any{
		"edit_ok":   true,
		"sync_done": true,
	})
	h.jobs.AppendLog(jobID, "system", "agent edit completed")
}

func agentEditOperationFromJob(job Job) agentEditOperation {
	out := agentEditOperation{
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
	out.EditOK, _ = job.Meta["edit_ok"].(bool)

	return out
}

var _ NodeAgentEditor = (*cursorNodeAgentEditor)(nil)
