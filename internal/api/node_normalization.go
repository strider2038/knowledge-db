package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/kb"
)

type normalizeOperation struct {
	ID          string    `json:"id"`
	NodePath    string    `json:"node_path"`
	Status      string    `json:"status"`
	Stage       string    `json:"stage"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
	SyncDone    bool      `json:"sync_done"`
	NormalizeOK bool      `json:"normalize_ok"`
}

type nodeNormalizer interface {
	NormalizeNode(ctx context.Context, dataPath string, node *kb.Node) error
}

type cursorNodeNormalizer struct {
}

func NewCursorNodeNormalizer() nodeNormalizer {
	return &cursorNodeNormalizer{}
}

func (n *cursorNodeNormalizer) NormalizeNode(ctx context.Context, dataPath string, node *kb.Node) error {
	if _, err := exec.LookPath("cursor-agent"); err != nil {
		return errors.Errorf("cursor-agent not found in PATH: %w", err)
	}

	prompt := buildNormalizationPrompt(node)
	cmd := exec.CommandContext(ctx, "cursor-agent", "--print", "--force", prompt)
	cmd.Dir = dataPath
	// Inherit server environment as-is (OAuth/session or explicit env outside app config).

	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Errorf("run cursor-agent: %w", err, errors.String("output", string(out)))
	}

	return nil
}

func buildNormalizationPrompt(node *kb.Node) string {
	var b strings.Builder
	b.WriteString("Нормализуй текущий markdown-узел базы знаний и сохрани изменения в этом же файле.\\n")
	b.WriteString("Сохрани фактический смысл, исправь форматирование markdown и frontmatter, не удаляй важные данные.\\n")
	b.WriteString("Не изменяй путь узла и не переименовывай файл.\\n")
	b.WriteString("Node path: ")
	b.WriteString(node.Path)
	b.WriteString("\\n\\n")
	b.WriteString("Frontmatter and content:\\n")
	metaJSON, _ := json.MarshalIndent(node.Metadata, "", "  ")
	b.WriteString("metadata: ")
	b.Write(metaJSON)
	b.WriteString("\\n\\nannotation:\\n")
	b.WriteString(node.Annotation)
	b.WriteString("\\n\\ncontent:\\n")
	b.WriteString(node.Content)

	return b.String()
}

// PostNodeNormalize обрабатывает POST /api/nodes/{path...}/normalize.
func (h *Handler) PostNodeNormalize(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.PathValue("path"), "/normalize")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	if h.normalizeRunner == nil {
		writeError(w, http.StatusServiceUnavailable, "node normalization unavailable")

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

	h.normalizeMu.RLock()
	for _, op := range h.normalizeOps {
		if op.NodePath == path && op.Status == "running" {
			h.normalizeMu.RUnlock()
			writeError(w, http.StatusConflict, "normalization already running for this node")

			return
		}
	}
	h.normalizeMu.RUnlock()

	op := normalizeOperation{
		ID:        uuid.NewString(),
		NodePath:  path,
		Status:    "running",
		Stage:     "normalize",
		StartedAt: time.Now().UTC(),
	}
	h.normalizeMu.Lock()
	h.normalizeOps[op.ID] = op
	h.normalizeMu.Unlock()

	go h.runNodeNormalization(context.Background(), op, node)

	writeJSON(w, op)
}

// GetNodeNormalizeStatus обрабатывает GET /api/node-normalization/{id}.
func (h *Handler) GetNodeNormalizeStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")

		return
	}
	h.normalizeMu.RLock()
	op, ok := h.normalizeOps[id]
	h.normalizeMu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "operation not found")

		return
	}
	writeJSON(w, op)
}

func (h *Handler) runNodeNormalization(ctx context.Context, op normalizeOperation, node *kb.Node) {
	if err := h.normalizeRunner.NormalizeNode(ctx, h.dataPath, node); err != nil {
		h.completeNormalizeOp(op.ID, "error", "normalize", err.Error(), false, false)

		return
	}

	h.updateNormalizeOpStage(op.ID, "sync")
	if h.gitDisabled || h.gitCommitter == nil {
		h.completeNormalizeOp(op.ID, "success", "done", "", true, false)

		return
	}

	if err := h.gitCommitter.Sync(ctx); err != nil {
		h.completeNormalizeOp(op.ID, "error", "sync", fmt.Sprintf("sync error: %v", err), true, false)

		return
	}

	h.completeNormalizeOp(op.ID, "success", "done", "", true, true)
}

func (h *Handler) updateNormalizeOpStage(id, stage string) {
	h.normalizeMu.Lock()
	defer h.normalizeMu.Unlock()
	op, ok := h.normalizeOps[id]
	if !ok {
		return
	}
	op.Stage = stage
	h.normalizeOps[id] = op
}

func (h *Handler) completeNormalizeOp(id, status, stage, errText string, normalizeOK, syncDone bool) {
	h.normalizeMu.Lock()
	defer h.normalizeMu.Unlock()
	op, ok := h.normalizeOps[id]
	if !ok {
		return
	}
	op.Status = status
	op.Stage = stage
	op.Error = errText
	op.NormalizeOK = normalizeOK
	op.SyncDone = syncDone
	op.FinishedAt = time.Now().UTC()
	h.normalizeOps[id] = op
	if status == "error" {
		clog.Error(context.Background(), "node normalize failed", "node_path", op.NodePath, "stage", stage, "error", errText)
	}
}

var _ nodeNormalizer = (*cursorNodeNormalizer)(nil)
