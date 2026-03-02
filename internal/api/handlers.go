package api

import (
	"encoding/json"
	"net/http"

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

// Static раздаёт embedded статику.
func (h *Handler) Static(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "" || path == "/" {
		path = "/index.html"
	}
	embedPath := "static" + path
	data, err := ui.Static.ReadFile(embedPath)
	if err != nil {
		if path != "/index.html" {
			data, err = ui.Static.ReadFile("static/index.html")
			if err == nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write(data)
				return
			}
		}
		http.NotFound(w, r)
		return
	}
	contentType := contentType(path)
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}

func contentType(path string) string {
	switch {
	case len(path) > 4 && path[len(path)-4:] == ".html":
		return "text/html; charset=utf-8"
	case len(path) > 3 && path[len(path)-3:] == ".js":
		return "application/javascript"
	case len(path) > 4 && path[len(path)-4:] == ".css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
