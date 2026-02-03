package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func Tracing(next http.Handler) http.Handler {
	tracer := otel.Tracer("mono-mvc/http")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.route", r.URL.Path),
		)

		sw, rw := ensureStatusWriter(w)
		next.ServeHTTP(rw, r.WithContext(ctx))

		if sw.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, "server error")
		} else {
			span.SetStatus(codes.Ok, "")
		}
	})
}
