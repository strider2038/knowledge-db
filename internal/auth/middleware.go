package auth

import (
	"net/http"
	"strings"

	"github.com/strider2038/knowledge-db/internal/auth/session"
)

// Middleware возвращает HTTP middleware для проверки сессии.
// Защищает все маршруты /api/* кроме /api/auth/* и allowlist (healthz, readyz).
// Применять только при включённой авторизации.
func Middleware(next http.Handler, store *session.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight должен проходить без проверки сессии
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)

			return
		}

		path := r.URL.Path
		if isAllowlisted(path) {
			next.ServeHTTP(w, r)

			return
		}

		// Защищённые маршруты: /api/* (кроме /api/auth/*)
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)

			return
		}

		cookie, err := r.Cookie(session.CookieName)
		if err != nil || cookie == nil || cookie.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if !store.Get(cookie.Value) {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowlisted(path string) bool {
	switch path {
	case "/healthz", "/readyz":
		return true
	}

	return strings.HasPrefix(path, "/api/auth/")
}
