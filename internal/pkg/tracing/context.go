package tracing

import (
	"context"

	"github.com/gofrs/uuid/v5"
)

type traceKey struct{}

// RequestID возвращает request_id из context или генерирует новый.
func RequestID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(traceKey{}).(uuid.UUID); ok && id != uuid.Nil {
		return id
	}

	return uuid.Must(uuid.NewV4())
}

// WithRequestID добавляет request_id в context.
func WithRequestID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}
