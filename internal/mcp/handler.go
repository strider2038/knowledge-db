package mcp

import (
	"net/http"
)

// Handler — заглушка для MCP endpoint /api/mcp.
type Handler struct {
	dataPath string
}

// NewHandler создаёт MCP handler.
func NewHandler(dataPath string) http.Handler {
	return &Handler{dataPath: dataPath}
}

// ServeHTTP обрабатывает запросы к /api/mcp (заглушка).
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"MCP not implemented"}`))
}
