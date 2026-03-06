package api

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
	"github.com/strider2038/knowledge-db/internal/ui"
)

// Handler — HTTP handlers для API.
type Handler struct {
	dataPath string
	ingester ingestion.Ingester
}

// NewHandler создаёт Handler.
func NewHandler(dataPath string, ingester ingestion.Ingester) *Handler {
	return &Handler{dataPath: dataPath, ingester: ingester}
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
			writeError(w, http.StatusNotFound, "node not found")

			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, node)
}

// GetTree обрабатывает GET /api/tree.
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	tree, err := kb.ReadTree(r.Context(), h.dataPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, tree)
}

// ListNodes обрабатывает GET /api/nodes (список узлов по path query).
func (h *Handler) ListNodes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	nodes, err := kb.ListNodes(r.Context(), h.dataPath, path)
	if err != nil {
		if errors.Is(err, kb.ErrNodeNotFound) {
			writeError(w, http.StatusNotFound, "path not found")

			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, map[string]any{"nodes": nodes})
}

// Search обрабатывает GET /api/search?q=... (заглушка).
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	_ = r.URL.Query().Get("q")
	writeJSON(w, map[string]any{"nodes": []any{}})
}

// Ingest обрабатывает POST /api/ingest.
func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")

		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")

		return
	}
	if req.Text == "" {
		writeError(w, http.StatusBadRequest, "text required")

		return
	}
	node, err := h.ingester.IngestText(r.Context(), req.Text)
	if err != nil {
		if errors.Is(err, ingestion.ErrNotImplemented) {
			writeError(w, http.StatusNotImplemented, "ingestion not implemented")

			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())

		return
	}
	writeJSON(w, node)
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
			fileServer.ServeHTTP(w, r)

			return
		}

		// Fallback для SPA (клиентский роутинг)
		serveIndexHTML(w, r, sub)
	}), nil
}

func isSPAClientRoute(path string) bool {
	path = strings.TrimPrefix(path, "/")

	return path == "add" || path == "search"
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
