package api

import (
	"net/http"
)

// NewMux создаёт ServeMux с зарегистрированными маршрутами (Go 1.22+).
func NewMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/nodes/{path...}", h.GetNode)
	mux.HandleFunc("GET /api/nodes", h.ListNodes)
	mux.HandleFunc("GET /api/tree", h.GetTree)
	mux.HandleFunc("GET /api/search", h.Search)
	mux.HandleFunc("POST /api/ingest", h.Ingest)
	mux.HandleFunc("GET /{path...}", h.Static)
	return mux
}
