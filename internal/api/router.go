package api

import (
	"net/http"

	"github.com/muonsoft/errors"
)

// NewMux создаёт ServeMux с зарегистрированными маршрутами (Go 1.22+).
func NewMux(h *Handler) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/nodes/{path...}", h.GetNode)
	mux.HandleFunc("GET /api/nodes", h.ListNodes)
	mux.HandleFunc("GET /api/tree", h.GetTree)
	mux.HandleFunc("GET /api/search", h.Search)
	mux.HandleFunc("POST /api/ingest", h.Ingest)
	spa, err := NewSPAHandler()
	if err != nil {
		return nil, errors.Errorf("new spa handler: %w", err)
	}
	mux.Handle("GET /{$}", spa)
	mux.Handle("GET /index.html", spa)
	mux.Handle("GET /{path...}", spa)
	return mux, nil
}
