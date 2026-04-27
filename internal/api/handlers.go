package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/muonsoft/clog"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/bootstrap/config"
	"github.com/strider2038/knowledge-db/internal/import/session"
	"github.com/strider2038/knowledge-db/internal/index"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	igit "github.com/strider2038/knowledge-db/internal/ingestion/git"
	"github.com/strider2038/knowledge-db/internal/ingestion/translationqueue"
	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/strider2038/knowledge-db/internal/pkg/urlutil"
	"github.com/strider2038/knowledge-db/internal/ui"
)

// Handler — HTTP handlers для API.
type Handler struct {
	dataPath          string
	uploadsDir        string
	ingester          ingestion.Ingester
	sessionStore      session.SessionStore
	translationQueue  *translationqueue.Queue
	gitCommitter      igit.GitCommitter
	commitMsgGen      *igit.CommitMessageGenerator
	gitDisabled       bool
	indexStore        *index.IndexStore
	syncWorker        *index.SyncWorker
	embeddingProvider index.EmbeddingProvider
	embeddingConfig   config.Embedding
	chatClient        chatResponsesClient
}

// NewHandler создаёт Handler.
func NewHandler(dataPath string, ingester ingestion.Ingester) *Handler {
	return &Handler{dataPath: dataPath, ingester: ingester}
}

// NewHandlerWithUploads создаёт Handler с поддержкой импорта (KB_UPLOADS_DIR).
// translationQueue — опционально; при nil endpoints перевода возвращают 503.
func NewHandlerWithUploads(dataPath, uploadsDir string, ingester ingestion.Ingester, translationQueue *translationqueue.Queue) *Handler {
	store := session.NewFileStore(uploadsDir, ingester)

	return &Handler{
		dataPath:         dataPath,
		uploadsDir:       uploadsDir,
		ingester:         ingester,
		sessionStore:     store,
		translationQueue: translationQueue,
	}
}

// SetGitCommitter устанавливает GitCommitter и CommitMessageGenerator.
// При gitDisabled=true git endpoints возвращают 503.
func (h *Handler) SetGitCommitter(committer igit.GitCommitter, msgGen *igit.CommitMessageGenerator, gitDisabled bool) {
	h.gitCommitter = committer
	h.commitMsgGen = msgGen
	h.gitDisabled = gitDisabled
}

// SetIndexComponents устанавливает компоненты индекса для RAG.
// При nil store все embedding endpoints возвращают 503.
func (h *Handler) SetIndexComponents(store *index.IndexStore, worker *index.SyncWorker, provider index.EmbeddingProvider, cfg config.Embedding) {
	h.indexStore = store
	h.syncWorker = worker
	h.embeddingProvider = provider
	h.embeddingConfig = cfg
	if cfg.IsConfigured() {
		chatURL, chatKey := cfg.ChatAPIConfig()
		h.chatClient = newOpenAIChatClient(chatURL, chatKey)
	}
}

// PostArticleTranslate обрабатывает POST /api/articles/translate/{path...}.
func (h *Handler) PostArticleTranslate(w http.ResponseWriter, r *http.Request) {
	h.handleArticleTranslate(w, r, true)
}

// GetArticleTranslate обрабатывает GET /api/articles/translate/{path...}.
func (h *Handler) GetArticleTranslate(w http.ResponseWriter, r *http.Request) {
	h.handleArticleTranslate(w, r, false)
}

func splitArticlePath(path string) (string, string) {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "", path
	}

	return path[:idx], path[idx+1:]
}

// GetNode обрабатывает GET /api/nodes/{path...}.
func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	node, err := kb.GetNode(r.Context(), h.dataPath, path)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			clog.Debug(r.Context(), "get node: not found", "path", path)
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		clog.Errorf(r.Context(), "get node: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, node)
}

// DeleteNode обрабатывает DELETE /api/nodes/{path...}.
func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	if err := kb.DeleteNode(r.Context(), h.dataPath, path); err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			clog.Debug(r.Context(), "delete node: not found", "path", path)
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		clog.Errorf(r.Context(), "delete node: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, map[string]any{"path": path, "deleted": true})
}

// MoveNode обрабатывает POST /api/nodes/{path...}/move (matched via POST /api/nodes/{path...}).
func (h *Handler) MoveNode(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	// Extract actual node path: expected pattern is "{nodePath}/move"
	nodePath, _ := strings.CutSuffix(path, "/move")
	if nodePath == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	var req struct {
		TargetPath string `json:"target_path"` //nolint:tagliatelle // REST API snake_case
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if req.TargetPath == "" {
		writeError(w, http.StatusBadRequest, "target_path is required")

		return
	}
	node, err := kb.MoveNode(r.Context(), h.dataPath, nodePath, req.TargetPath)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			clog.Debug(r.Context(), "move node: not found", "path", nodePath)
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		if errors.Is(err, kb.ErrConflict) {
			clog.Debug(r.Context(), "move node: conflict", "target_path", req.TargetPath)
			writeError(w, http.StatusConflict, "target path already exists")

			return
		}
		if errors.Is(err, kb.ErrInvalidPath) {
			clog.Debug(r.Context(), "move node: invalid target_path", "target_path", req.TargetPath)
			writeError(w, http.StatusBadRequest, "invalid target_path")

			return
		}
		clog.Errorf(r.Context(), "move node: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, node)
}

// PatchNode обрабатывает PATCH /api/nodes/{path...} — обновление поля manual_processed в frontmatter.
func (h *Handler) PatchNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if len(raw) != 1 {
		writeError(w, http.StatusBadRequest, "body must contain only manual_processed")

		return
	}
	rawVal, ok := raw["manual_processed"]
	if !ok {
		writeError(w, http.StatusBadRequest, "body must contain only manual_processed")

		return
	}
	var value bool
	if err := json.Unmarshal(rawVal, &value); err != nil {
		writeError(w, http.StatusBadRequest, "manual_processed must be a boolean")

		return
	}
	if err := kb.PatchNodeManualProcessed(r.Context(), h.dataPath, path, value); err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			clog.Debug(r.Context(), "patch node: not found", "path", path)
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		clog.Errorf(r.Context(), "patch node: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	node, err := kb.GetNode(r.Context(), h.dataPath, path)
	if err != nil {
		clog.Errorf(r.Context(), "patch node reload: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, node)
}

// GetTree обрабатывает GET /api/tree.
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	tree, err := kb.ReadTree(r.Context(), h.dataPath)
	if err != nil {
		clog.Errorf(r.Context(), "get tree: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, tree)
}

// ListNodes обрабатывает GET /api/nodes (список узлов по path query).
// При recursive=true возвращает {nodes: NodeListItem[], total: number}.
// При recursive=false — обратная совместимость: {nodes: TreeNode[]}.
func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path := q.Get("path")
	recursive, _ := strconv.ParseBool(q.Get("recursive"))

	if !recursive {
		nodes, err := kb.ListNodes(r.Context(), h.dataPath, path)
		if err != nil {
			if errors.Is(err, kb.ErrNodeNotFound) {
				clog.Debug(r.Context(), "list nodes: path not found", "path", path)
				writeError(w, http.StatusNotFound, "path not found")

				return
			}
			clog.Errorf(r.Context(), "list nodes: %w", err)
			writeError(w, http.StatusInternalServerError, err.Error())

			return
		}
		writeJSON(w, map[string]any{"nodes": nodes})

		return
	}

	opts := kb.ListNodesOptions{
		Path:      path,
		Recursive: true,
		Q:         q.Get("q"),
		Sort:      q.Get("sort"),
		Order:     q.Get("order"),
	}
	if opts.Sort == "" {
		opts.Sort = "title"
	}
	if opts.Order == "" {
		opts.Order = "asc"
	}
	if limit, err := strconv.Atoi(q.Get("limit")); err == nil && limit > 0 {
		opts.Limit = limit
	} else {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	if offset, err := strconv.Atoi(q.Get("offset")); err == nil && offset >= 0 {
		opts.Offset = offset
	}
	if typeParam := q.Get("type"); typeParam != "" {
		for t := range strings.SplitSeq(typeParam, ",") {
			if s := strings.TrimSpace(t); s != "" {
				opts.Types = append(opts.Types, s)
			}
		}
	}
	switch strings.TrimSpace(q.Get("manual_processed")) {
	case "":
		// no filter
	case "true":
		v := true
		opts.ManualProcessed = &v
	case "false":
		v := false
		opts.ManualProcessed = &v
	default:
		writeError(w, http.StatusBadRequest, "invalid manual_processed, expected true or false")

		return
	}

	nodes, total, err := kb.ListNodesWithOptions(r.Context(), h.dataPath, opts)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			clog.Debug(r.Context(), "list nodes: path not found", "path", opts.Path)
			writeError(w, http.StatusNotFound, "path not found")

			return
		}
		clog.Errorf(r.Context(), "list nodes: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, map[string]any{"nodes": nodes, "total": total})
}

// GetAsset обрабатывает GET /api/assets/{path...} — раздаёт файлы из базы (изображения, вложения).
func (h *Handler) GetAsset(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	fullPath := filepath.Join(h.dataPath, clean)
	if rel, err := filepath.Rel(h.dataPath, fullPath); err != nil || strings.HasPrefix(rel, "..") {
		writeError(w, http.StatusBadRequest, "invalid path")

		return
	}
	http.ServeFile(w, r, fullPath)
}

// Search обрабатывает GET /api/search?q=... (заглушка).
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	_ = r.URL.Query().Get("q")
	writeJSON(w, map[string]any{"nodes": []any{}})
}

// GetGitStatus обрабатывает GET /api/git/status.
func (h *Handler) GetGitStatus(w http.ResponseWriter, r *http.Request) {
	if h.gitDisabled || h.gitCommitter == nil {
		writeError(w, http.StatusServiceUnavailable, "git is disabled")

		return
	}
	status, err := h.gitCommitter.Status(r.Context())
	if err != nil {
		clog.Errorf(r.Context(), "git status: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, map[string]any{
		"has_changes":   status.HasChanges,
		"changed_files": status.ChangedFiles,
	})
}

// PostGitCommit обрабатывает POST /api/git/commit.
func (h *Handler) PostGitCommit(w http.ResponseWriter, r *http.Request) {
	if h.gitDisabled || h.gitCommitter == nil {
		writeError(w, http.StatusServiceUnavailable, "git is disabled")

		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}

	status, err := h.gitCommitter.Status(r.Context())
	if err != nil {
		clog.Errorf(r.Context(), "git commit: status check: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	if !status.HasChanges {
		writeJSON(w, map[string]any{"committed": false, "message": "no changes to commit"})

		return
	}

	message := req.Message
	if message == "" {
		diffStat, diffErr := h.gitCommitter.DiffStat(r.Context())
		if diffErr != nil {
			clog.Warn(r.Context(), "git commit: diff stat error, using fallback", "error", diffErr)
			diffStat = ""
		}
		message = h.commitMsgGen.Generate(r.Context(), diffStat)
	}

	if err := h.gitCommitter.CommitAll(r.Context(), message); err != nil {
		clog.Errorf(r.Context(), "git commit: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	clog.Info(r.Context(), "git commit: success", "message", message)
	writeJSON(w, map[string]any{"message": message, "committed": true})
}

// Ingest обрабатывает POST /api/ingest.
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}
	var req struct {
		Text         string `json:"text"`
		SourceURL    string `json:"source_url"`    //nolint:tagliatelle // REST API snake_case
		SourceAuthor string `json:"source_author"` //nolint:tagliatelle // REST API snake_case
		TypeHint     string `json:"type_hint"`     //nolint:tagliatelle // REST API snake_case
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text required")

		return
	}
	sourceURL := req.SourceURL
	if sourceURL != "" {
		sourceURL = urlutil.StripTrackingParamsFromURL(sourceURL)
	}
	typeHint := req.TypeHint
	if typeHint != "" && typeHint != "auto" && typeHint != "article" && typeHint != "link" && typeHint != "note" {
		typeHint = ""
	}
	node, err := h.ingester.IngestText(r.Context(), ingestion.IngestRequest{
		Text:         req.Text,
		SourceURL:    sourceURL,
		SourceAuthor: req.SourceAuthor,
		TypeHint:     typeHint,
	})
	if err != nil {
		if errors.Is(err, ingestion.ErrNotImplemented) {
			clog.Warn(r.Context(), "ingest: not implemented")
			writeError(w, http.StatusNotImplemented, "ingestion not implemented")

			return
		}
		clog.Errorf(r.Context(), "ingest: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	clog.Info(r.Context(), "ingest: complete", "path", node.Path)
	writeJSON(w, node)
}

func (h *Handler) handleArticleTranslate(w http.ResponseWriter, r *http.Request, isPost bool) {
	if h.translationQueue == nil {
		writeError(w, http.StatusServiceUnavailable, "translation service unavailable")

		return
	}

	path := r.PathValue("path")
	if path == "" {
		writeError(w, http.StatusBadRequest, "path required")

		return
	}

	node, err := kb.GetNode(r.Context(), h.dataPath, path)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			clog.Debug(r.Context(), "article translate: node not found", "path", path)
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		clog.Errorf(r.Context(), "article translate: get node: %w", err)
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}

	nodeType, _ := node.Metadata["type"].(string)
	if nodeType != "article" {
		clog.Debug(r.Context(), "article translate: not an article", "path", path)
		writeError(w, http.StatusBadRequest, "node is not an article")

		return
	}

	themePath, slug := splitArticlePath(path)
	translationPath := themePath + "/" + slug + ".ru"

	if _, err := kb.GetNode(r.Context(), h.dataPath, translationPath); err == nil {
		writeJSON(w, map[string]any{"status": translationqueue.StatusDone})

		return
	}

	status, errMsg := h.translationQueue.GetStatus(themePath, slug)
	if status == translationqueue.StatusPending || status == translationqueue.StatusInProgress {
		writeJSON(w, map[string]any{"status": status})

		return
	}

	if isPost {
		status, _ = h.translationQueue.Enqueue(themePath, slug)
		clog.Info(r.Context(), "translation: enqueued", "theme", themePath, "slug", slug, "status", status)
	}

	resp := map[string]any{"status": status}
	if status == translationqueue.StatusFailed && errMsg != "" {
		resp["error"] = errMsg
	}
	writeJSON(w, resp)
}

// NewSPAHandler создаёт handler для раздачи embedded статики (FileServer + SPA fallback).
func NewSPAHandler() (http.Handler, error) {
	sub, err := fs.Sub(ui.Static, "static")
	if err != nil {
		return nil, errors.Errorf("ui static: %w", err)
	}
	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// SPA-маршруты (add, search) — index.html
		if isSPAClientRoute(path) {
			serveIndexHTML(w, r, sub)

			return
		}

		// Файл существует — FileServer
		trimmed := strings.TrimPrefix(path, "/")
		if _, err := sub.Open(trimmed); err == nil {
			// Хешированные assets — immutable, index.html — no-cache
			if trimmed == "index.html" {
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			} else if strings.HasPrefix(trimmed, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)

			return
		}

		// /assets/* — статика; 404 вместо index.html (иначе MIME type error)
		if strings.HasPrefix(trimmed, "assets/") {
			http.NotFound(w, r)

			return
		}

		// Fallback для SPA (клиентский роутинг)
		serveIndexHTML(w, r, sub)
	}), nil
}

func isSPAClientRoute(path string) bool {
	path = strings.TrimPrefix(path, "/")

	return path == "add" || path == "search" || path == "chat"
}

// serveIndexHTML отдаёт index.html без FileServer, чтобы избежать редиректов.
func serveIndexHTML(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	const indexFile = "index.html"
	file, err := fsys.Open(indexFile)
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)

		return
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "cannot stat index.html", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	reader, ok := file.(io.ReadSeeker)
	if !ok {
		http.Error(w, "cannot read index.html", http.StatusInternalServerError)

		return
	}
	http.ServeContent(w, r, indexFile, stat.ModTime(), reader)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
	}
}
