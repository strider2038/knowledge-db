package api

import (
	"net/http"

	"github.com/muonsoft/errors"
)

// NewMux создаёт ServeMux с зарегистрированными маршрутами (Go 1.22+).
// auth — опционально; при nil auth endpoints не регистрируются.
func NewMux(h *Handler, auth *AuthHandler) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	if auth != nil {
		mux.HandleFunc("POST /api/auth/login", auth.Login)
		mux.HandleFunc("GET /api/auth/session", auth.Session)
		mux.HandleFunc("POST /api/auth/logout", auth.Logout)
		mux.HandleFunc("GET /api/auth/google", auth.GoogleOAuthStart)
		mux.HandleFunc("GET /api/auth/google/callback", auth.GoogleOAuthCallback)
	}
	mux.HandleFunc("GET /api/nodes/{path...}", h.GetNode)
	mux.HandleFunc("GET /api/nodes", h.ListNodes)
	mux.HandleFunc("GET /api/assets/{path...}", h.GetAsset)
	mux.HandleFunc("GET /api/tree", h.GetTree)
	mux.HandleFunc("GET /api/search", h.Search)
	mux.HandleFunc("POST /api/ingest", h.Ingest)
	mux.HandleFunc("POST /api/import/telegram", h.ImportTelegramCreate)
	mux.HandleFunc("GET /api/import/telegram/session/{id}", h.ImportTelegramGetSession)
	mux.HandleFunc("POST /api/import/telegram/session/{id}/accept", h.ImportTelegramAccept)
	mux.HandleFunc("POST /api/import/telegram/session/{id}/reject", h.ImportTelegramReject)
	mux.HandleFunc("POST /api/articles/translate/{path...}", h.PostArticleTranslate)
	mux.HandleFunc("GET /api/articles/translate/{path...}", h.GetArticleTranslate)
	spa, err := NewSPAHandler()
	if err != nil {
		return nil, errors.Errorf("new spa handler: %w", err)
	}
	mux.Handle("GET /{$}", spa)
	mux.Handle("GET /index.html", spa)
	mux.Handle("GET /{path...}", spa)

	return mux, nil
}
