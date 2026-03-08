package api

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// Gzip middleware сжимает ответы при наличии Accept-Encoding: gzip.
// Не сжимает при Range-запросах (частичная загрузка несовместима с gzip).
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)

			return
		}
		if r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)

			return
		}

		gz := gzip.NewWriter(w)
		defer func() { _ = gz.Close() }()

		w.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(&gzipResponseWriter{w: w, gz: gz}, r)
	})
}

type gzipResponseWriter struct {
	w  http.ResponseWriter
	gz *gzip.Writer
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	// Content-Length от внутреннего handler — для несжатых данных.
	// При gzip размер меняется, убираем чтобы использовать chunked transfer.
	w.Header().Del("Content-Length")
	w.w.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.gz.Write(b)
}

var _ http.ResponseWriter = (*gzipResponseWriter)(nil)
