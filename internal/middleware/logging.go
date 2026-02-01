package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw, rw := ensureStatusWriter(w)

			next.ServeHTTP(rw, r)

			spanCtx := trace.SpanFromContext(r.Context()).SpanContext()
			traceID := ""
			if spanCtx.IsValid() {
				traceID = spanCtx.TraceID().String()
			}

			logger.Info("request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Int("bytes", sw.bytes),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("request_id", GetRequestID(r.Context())),
				slog.String("trace_id", traceID),
				slog.String("remote_ip", r.RemoteAddr),
			)
		})
	}
}
