// Package middleware hosts request_id, recoverer, logger, and timeout
// middleware used by the chi router.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/kevin907/zookeeper/internal/platform/logging"
)

// RequestLogger is a slog-backed request logger: it logs method, path,
// status, bytes, request id, and duration at the response boundary.
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rid := middleware.GetReqID(r.Context())
			logger := base.With("request_id", rid)
			ctx := logging.WithLogger(r.Context(), logger)

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r.WithContext(ctx))

			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
