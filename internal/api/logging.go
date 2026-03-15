package api

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/muonsoft/clog"

	"github.com/strider2038/knowledge-db/internal/pkg/tracing"
)

// LoggingMiddleware обогащает context логгером с request_id, request_method, request_url.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := tracing.RequestID(r.Context())

		logger := slog.Default().With(
			slog.String("runnable", "httpserver"),
			slog.String("request_id", requestID.String()),
			slog.String("request_method", r.Method),
			slog.String("request_url", r.URL.Path),
		)

		ctx := clog.NewContext(r.Context(), logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestLoggingMiddleware логирует "request {method} {path} started" и "completed" с latency и status.
func RequestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startAt := time.Now()
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		logger := clog.FromContext(r.Context()).With(
			slog.String("request_user_agent", r.UserAgent()),
			slog.String("request_referrer", r.Referer()),
			slog.String("request_host", host),
		)

		message := fmt.Sprintf("request %s %s", r.Method, r.URL.Path)
		logger.Info(message + " started")

		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			logger.
				With(
					slog.Int64("request_latency_ms", time.Since(startAt).Milliseconds()),
					slog.Int("request_status", rec.status),
				).
				Info(message + " completed")
		}()

		next.ServeHTTP(rec, r)
	})
}

type responseRecorder struct {
	http.ResponseWriter

	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
