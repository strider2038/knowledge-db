package tracing

import (
	"net/http"

	"github.com/gofrs/uuid/v5"
)

const Header = "X-Request-Id"

// Middleware читает X-Request-Id из заголовка или генерирует UUID, кладёт в context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, err := uuid.FromString(r.Header.Get(Header))
		if err != nil || requestID == uuid.Nil {
			requestID = uuid.Must(uuid.NewV7())
		}

		ctx := WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
