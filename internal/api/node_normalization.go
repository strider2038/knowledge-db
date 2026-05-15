package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
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

type cursorNodeNormalizer struct {
}

type cursorStreamState struct {
	assistantText strings.Builder
}

func NewCursorNodeNormalizer() nodeNormalizer {
	return &cursorNodeNormalizer{}
}

func (n *cursorNodeNormalizer) NormalizeNode(ctx context.Context, dataPath string, node *kb.Node, onLog func(stream, text string)) error {
	if _, err := exec.LookPath("cursor-agent"); err != nil {
		return errors.Errorf("cursor-agent not found in PATH: %w", err)
	}

	prompt := buildNormalizationPrompt(node)
	cmd := exec.CommandContext(
		ctx,
		"cursor-agent",
		"--print",
		"--output-format", "stream-json",
		"--force",
		prompt,
	)
	cmd.Dir = dataPath
	// Inherit server environment as-is (OAuth/session or explicit env outside app config).

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return errors.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return errors.Errorf("start cursor-agent: %w", err)
	}
	state := &cursorStreamState{}
	readPipe := func(stream string, r io.Reader, wg *sync.WaitGroup) {
		defer wg.Done()
		s := bufio.NewScanner(r)
		s.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
		for s.Scan() {
			for _, line := range parseCursorLogEvents(s.Text(), state) {
				onLog(stream, line)
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go readPipe("stdout", stdoutPipe, &wg)
	go readPipe("stderr", stderrPipe, &wg)
	err = cmd.Wait()
	wg.Wait()
	if text := strings.TrimSpace(state.assistantText.String()); text != "" {
		onLog("stdout", text)
	}
	if err != nil {
		return errors.Errorf("run cursor-agent: %w", err)
	}

	return nil
}

func parseCursorLogEvents(line string, state *cursorStreamState) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return []string{line}
	}
	typeVal, _ := obj["type"].(string)
	subtype, _ := obj["subtype"].(string)

	if thinking := parseThinkingEvent(typeVal, subtype); thinking != nil {
		return thinking
	}
	if typeVal == "assistant" {
		collectAssistantText(state, obj)

		return nil
	}
	if typeVal == "tool_call" {
		return []string{formatToolCallEvent(obj, subtype)}
	}
	if typeVal == "result" {
		if result, ok := obj["result"].(string); ok && result != "" {
			// If we already accumulated the same text via assistant deltas, don't duplicate.
			if strings.TrimSpace(result) != strings.TrimSpace(state.assistantText.String()) {
				return []string{result}
			}

			return nil
		}

		return []string{"result:" + subtype}
	}

	if subtype != "" {
		return []string{typeVal + ":" + subtype}
	}

	return []string{typeVal + " " + compactKV(obj)}
}

func parseThinkingEvent(typeVal, subtype string) []string {
	if typeVal != "thinking" {
		return nil
	}
	if subtype == "completed" {
		return []string{"thinking completed"}
	}

	return []string{}
}

func collectAssistantText(state *cursorStreamState, obj map[string]any) {
	msg, ok := obj["message"].(map[string]any)
	if !ok {
		return
	}
	content, ok := msg["content"].([]any)
	if !ok {
		return
	}
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		text, ok := m["text"].(string)
		if ok && text != "" {
			state.assistantText.WriteString(text)
		}
	}
}

func formatToolCallEvent(obj map[string]any, subtype string) string {
	if toolCall, ok := obj["tool_call"].(map[string]any); ok {
		for toolName := range toolCall {
			if subtype != "" {
				return "tool:" + toolName + ":" + subtype
			}

			return "tool:" + toolName
		}
	}
	if subtype != "" {
		return "tool_call:" + subtype
	}

	return "tool_call"
}

func compactKV(obj map[string]any) string {
	parts := make([]string, 0, len(obj))
	for k, v := range obj {
		if k == "message" || k == "result" || k == "content" {
			continue
		}
		parts = append(parts, k+"="+valueToString(v))
	}

	return strings.Join(parts, " ")
}

func valueToString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}

		return "false"
	default:
		if vv := fmt.Sprintf("%T", v); strings.HasPrefix(vv, "[]") {
			return "[...]"
		}

		return "{...}"
	}
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
	if _, ok := h.normalizeRunner.(*cursorNodeNormalizer); ok {
		if _, err := exec.LookPath("cursor-agent"); err != nil {
			writeError(w, http.StatusServiceUnavailable, "cursor-agent not found in PATH")

			return
		}
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
	h.appendNormalizeLogLocked(&op, "system", "normalization started")
	h.normalizeOps[op.ID] = op
	h.normalizeMu.Unlock()

	go h.runNodeNormalization(context.WithoutCancel(r.Context()), op, node)

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
	h.normalizeMu.RLock()
	op, ok := h.normalizeOps[id]
	h.normalizeMu.RUnlock()
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
	writeJSON(w, normalizeLogsResponse{Entries: entries, NextOffset: op.NextOffset})
}

func (h *Handler) runNodeNormalization(ctx context.Context, op normalizeOperation, node *kb.Node) {
	if err := h.normalizeRunner.NormalizeNode(ctx, h.dataPath, node, func(stream, text string) {
		h.appendNormalizeLog(op.ID, stream, text)
	}); err != nil {
		h.completeNormalizeOp(ctx, op.ID, "error", "normalize", err.Error(), false, false)

		return
	}

	h.updateNormalizeOpStage(op.ID, "sync")
	if h.gitDisabled || h.gitCommitter == nil {
		h.completeNormalizeOp(ctx, op.ID, "success", "done", "", true, false)

		return
	}

	if err := h.gitCommitter.Sync(ctx); err != nil {
		h.completeNormalizeOp(ctx, op.ID, "error", "sync", fmt.Sprintf("sync error: %v", err), true, false)

		return
	}

	h.completeNormalizeOp(ctx, op.ID, "success", "done", "", true, true)
}

func (h *Handler) appendNormalizeLog(id, stream, text string) {
	h.normalizeMu.Lock()
	defer h.normalizeMu.Unlock()
	op, ok := h.normalizeOps[id]
	if !ok {
		return
	}
	h.appendNormalizeLogLocked(&op, stream, text)
	h.normalizeOps[id] = op
}

func (h *Handler) appendNormalizeLogLocked(op *normalizeOperation, stream, text string) {
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

func (h *Handler) completeNormalizeOp(ctx context.Context, id, status, stage, errText string, normalizeOK, syncDone bool) {
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
	finishedAt := time.Now().UTC()
	op.FinishedAt = &finishedAt
	if status == "success" {
		h.appendNormalizeLogLocked(&op, "system", "normalization completed")
	} else {
		h.appendNormalizeLogLocked(&op, "system", "normalization failed: "+errText)
	}
	h.normalizeOps[id] = op
	if status == "error" {
		clog.Error(ctx, "node normalize failed", "node_path", op.NodePath, "stage", stage, "error", errText)
	}
}

var _ nodeNormalizer = (*cursorNodeNormalizer)(nil)
