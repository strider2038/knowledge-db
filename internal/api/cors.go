package api

import (
	"net/http"
)

// CORS middleware добавляет CORS-заголовки для cross-origin запросов (dev: Vite на :5173).
// origin — разрешённый origin (например "http://localhost:5173" или "*").
func CORS(next http.Handler, origin string) http.Handler {
	if origin == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}
		next.ServeHTTP(w, r)
	})
}
