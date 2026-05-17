package api

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
	"github.com/strider2038/knowledge-db/internal/kb"
)

const (
	jobStatusQueued  = "queued"
	jobStatusRunning = "running"
	jobStatusSuccess = "success"
	jobStatusError   = "error"

	jobTypeNormalize   = "node_normalize"
	jobTypeDumpImages  = "node_dump_images"
	jobTypeRefreshDesc = "refresh_description"
	jobTypeTranslate   = "article_translate"
)

var (
	errJobPathRequired            = errors.New("path required")
	errJobTypeTargetRequired      = errors.New("type and target are required")
	errJobUnsupportedType         = errors.New("unsupported job type")
	errJobDescRefreshUnavailable  = errors.New("description refresh unavailable")
	errJobNormalizeUnavailable    = errors.New("node normalization unavailable")
	errJobCursorAgentUnavailable  = errors.New("cursor-agent not found in PATH")
	errJobNodeNotArticle          = errors.New("node is not an article")
	errJobNormalizeAlreadyRunning = errors.New("normalization already running for this node")
	errJobDumpAlreadyRunning      = errors.New("dump images already running for this node")
	errJobRefreshAlreadyRunning   = errors.New("refresh description already running for this node")
	errJobTranslateUnavailable    = errors.New("translation service unavailable")
)

type Job struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Target     string         `json:"target"`
	Status     string         `json:"status"`
	Stage      string         `json:"stage"`
	Error      string         `json:"error,omitempty"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt *time.Time     `json:"finished_at,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
	NextOffset int64          `json:"next_offset"`
	Logs       []JobLogEntry  `json:"-"`
}

type JobLogEntry struct {
	Offset    int64     `json:"offset"`
	Stream    string    `json:"stream"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type JobLogsResponse struct {
	Entries    []JobLogEntry `json:"entries"`
	NextOffset int64         `json:"next_offset"`
}

type JobManager struct {
	mu   sync.RWMutex
	jobs map[string]Job
}

func NewJobManager() *JobManager {
	return &JobManager{jobs: make(map[string]Job)}
}

func (m *JobManager) Create(jobType, target, stage string, meta map[string]any) Job {
	if stage == "" {
		stage = "start"
	}
	now := time.Now().UTC()
	job := Job{
		ID:        uuid.NewString(),
		Type:      strings.TrimSpace(jobType),
		Target:    strings.TrimSpace(target),
		Status:    jobStatusQueued,
		Stage:     stage,
		StartedAt: now,
		Meta:      cloneMeta(meta),
	}
	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	return cloneJob(job)
}

func (m *JobManager) Get(id string) (Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]

	return cloneJob(job), ok
}

func (m *JobManager) SetRunning(id, stage string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	job.Status = jobStatusRunning
	if stage != "" {
		job.Stage = stage
	}
	m.jobs[id] = job
}

func (m *JobManager) SetStage(id, stage string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	job.Stage = stage
	m.jobs[id] = job
}

func (m *JobManager) AppendLog(id, stream, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	job.NextOffset++
	job.Logs = append(job.Logs, JobLogEntry{
		Offset:    job.NextOffset,
		Stream:    stream,
		Text:      text,
		Timestamp: time.Now().UTC(),
	})
	if len(job.Logs) > 1000 {
		job.Logs = append([]JobLogEntry(nil), job.Logs[len(job.Logs)-1000:]...)
	}
	m.jobs[id] = job
}

func (m *JobManager) CompleteSuccess(id, stage string, metaPatch map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	now := time.Now().UTC()
	job.Status = jobStatusSuccess
	if stage != "" {
		job.Stage = stage
	}
	job.FinishedAt = &now
	mergeMeta(job.Meta, metaPatch)
	m.jobs[id] = job
}

func (m *JobManager) CompleteError(id, stage, errText string, metaPatch map[string]any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	now := time.Now().UTC()
	job.Status = jobStatusError
	if stage != "" {
		job.Stage = stage
	}
	job.Error = errText
	job.FinishedAt = &now
	mergeMeta(job.Meta, metaPatch)
	m.jobs[id] = job
}

func (m *JobManager) GetLogs(id string, after int64) (JobLogsResponse, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return JobLogsResponse{}, false
	}
	entries := make([]JobLogEntry, 0)
	for _, entry := range job.Logs {
		if entry.Offset > after {
			entries = append(entries, entry)
		}
	}

	return JobLogsResponse{Entries: entries, NextOffset: job.NextOffset}, true
}

func (m *JobManager) FindRunning(jobType, target string) (Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, job := range m.jobs {
		if job.Type == jobType && job.Target == target && (job.Status == jobStatusQueued || job.Status == jobStatusRunning) {
			return cloneJob(job), true
		}
	}

	return Job{}, false
}

func cloneMeta(meta map[string]any) map[string]any {
	if meta == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(meta))
	maps.Copy(out, meta)

	return out
}

func cloneJob(job Job) Job {
	job.Meta = cloneMeta(job.Meta)
	if job.Logs != nil {
		job.Logs = append([]JobLogEntry(nil), job.Logs...)
	}

	return job
}

func mergeMeta(dst map[string]any, patch map[string]any) {
	if dst == nil || patch == nil {
		return
	}
	maps.Copy(dst, patch)
}

func (h *Handler) PostJobs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type    string         `json:"type"`
		Target  string         `json:"target"`
		Options map[string]any `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	req.Type = strings.TrimSpace(req.Type)
	req.Target = strings.TrimSpace(req.Target)
	if req.Type == "" || req.Target == "" {
		writeError(w, http.StatusBadRequest, errJobTypeTargetRequired.Error())

		return
	}

	switch req.Type {
	case jobTypeNormalize:
		job, err := h.startNormalizeJob(r.Context(), req.Target)
		if err != nil {
			writeError(w, httpStatusFromJobErr(err), err.Error())

			return
		}
		writeJSON(w, job)
	case jobTypeDumpImages:
		job, err := h.startDumpImagesJob(r.Context(), req.Target)
		if err != nil {
			writeError(w, httpStatusFromJobErr(err), err.Error())

			return
		}
		writeJSON(w, job)
	case jobTypeRefreshDesc:
		job, err := h.startRefreshDescriptionJob(r.Context(), req.Target)
		if err != nil {
			writeError(w, httpStatusFromJobErr(err), err.Error())

			return
		}
		writeJSON(w, job)
	case jobTypeTranslate:
		job, err := h.startTranslateJob(r.Context(), req.Target)
		if err != nil {
			writeError(w, httpStatusFromJobErr(err), err.Error())

			return
		}
		writeJSON(w, job)
	default:
		writeError(w, http.StatusBadRequest, errJobUnsupportedType.Error())
	}
}

func (h *Handler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")

		return
	}
	job, ok := h.jobs.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")

		return
	}
	writeJSON(w, job)
}

func (h *Handler) GetJobLogs(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
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
		writeError(w, http.StatusNotFound, "job not found")

		return
	}
	writeJSON(w, resp)
}

func httpStatusFromJobErr(err error) int {
	switch {
	case errors.Is(err, kb.ErrNodeNotFound):
		return http.StatusNotFound
	case errors.Is(err, ingestion.ErrSourceURLRequired):
		return http.StatusBadRequest
	case errors.Is(err, errJobNormalizeUnavailable), errors.Is(err, errJobCursorAgentUnavailable), errors.Is(err, errJobDescRefreshUnavailable), errors.Is(err, errJobTranslateUnavailable):
		return http.StatusServiceUnavailable
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "unsupported") || strings.Contains(err.Error(), "not an article") {
			return http.StatusBadRequest
		}
		if strings.Contains(err.Error(), "already running") {
			return http.StatusConflict
		}

		return http.StatusBadGateway
	}
}

func (h *Handler) startRefreshDescriptionJob(ctx context.Context, path string) (Job, error) {
	if path == "" {
		return Job{}, errJobPathRequired
	}
	refresher, ok := h.ingester.(ingestion.DescriptionRefresher)
	if !ok {
		return Job{}, errJobDescRefreshUnavailable
	}
	if running, ok := h.jobs.FindRunning(jobTypeRefreshDesc, path); ok {
		return running, errJobRefreshAlreadyRunning
	}
	job := h.jobs.Create(jobTypeRefreshDesc, path, "fetch", map[string]any{
		"node_path": path,
	})
	h.jobs.SetRunning(job.ID, "fetch")
	h.jobs.AppendLog(job.ID, "system", "refresh description started")
	go func(jobID string) {
		h.jobs.AppendLog(jobID, "system", "stage: llm")
		node, err := refresher.RefreshDescription(context.WithoutCancel(ctx), path)
		if err != nil {
			h.jobs.CompleteError(jobID, "refresh", err.Error(), nil)
			h.jobs.AppendLog(jobID, "system", "refresh description failed: "+err.Error())

			return
		}
		h.jobs.SetStage(jobID, "sync")
		if h.syncWorker != nil {
			//nolint:contextcheck // sync worker API accepts event-only contract
			h.syncWorker.Send(index.SingleNodeEvent{Path: node.Path})
		}
		h.jobs.CompleteSuccess(jobID, "done", map[string]any{
			"result_path": node.Path,
		})
		h.jobs.AppendLog(jobID, "system", "refresh description completed")
	}(job.ID)

	return job, nil
}

func (h *Handler) startTranslateJob(ctx context.Context, path string) (Job, error) {
	if h.translationQueue == nil {
		return Job{}, errJobTranslateUnavailable
	}
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
	if running, ok := h.jobs.FindRunning(jobTypeTranslate, path); ok {
		return running, nil
	}
	themePath, slug := splitArticlePath(path)
	translationPath := themePath + "/" + slug + ".ru"
	job := h.jobs.Create(jobTypeTranslate, path, "queue", map[string]any{
		"node_path": path,
	})
	h.jobs.SetRunning(job.ID, "queue")
	h.jobs.AppendLog(job.ID, "system", "translation job started")
	//nolint:contextcheck // translation queue is intentionally decoupled from request cancellation
	go func(jobID string) {
		if _, err := kb.GetNode(context.Background(), h.dataPath, translationPath); err == nil {
			h.jobs.CompleteSuccess(jobID, "done", nil)
			h.jobs.AppendLog(jobID, "system", "translation already exists")

			return
		}
		status, _ := h.translationQueue.Enqueue(themePath, slug)
		h.jobs.AppendLog(jobID, "system", "translation enqueued: "+status)
		ticker := time.NewTicker(1500 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			st, errMsg := h.translationQueue.GetStatus(themePath, slug)
			h.jobs.SetStage(jobID, st)
			if st == translationqueue.StatusDone {
				h.jobs.CompleteSuccess(jobID, "done", nil)
				h.jobs.AppendLog(jobID, "system", "translation completed")

				return
			}
			if st == translationqueue.StatusFailed {
				if errMsg == "" {
					errMsg = "translation failed"
				}
				h.jobs.CompleteError(jobID, "failed", errMsg, nil)
				h.jobs.AppendLog(jobID, "system", "translation failed: "+errMsg)

				return
			}
		}
	}(job.ID)

	return job, nil
}
