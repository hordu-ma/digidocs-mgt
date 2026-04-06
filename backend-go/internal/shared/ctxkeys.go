package shared

import "context"

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDFromContext extracts the request ID stored by the RequestID middleware.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

// WithRequestID returns a copy of ctx with the request ID attached.
func WithRequestID(ctx context.Context, rid string) context.Context {
	return context.WithValue(ctx, requestIDKey, rid)
}
