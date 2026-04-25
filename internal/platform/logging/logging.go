// Package logging constructs the application slog.Logger and provides
// typed context accessors for request-scoped logging.
package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

type ctxKey struct{}

// New returns a JSON-handler slog.Logger configured for the given level
// string (debug | info | warn | error). Unknown values default to info.
func New(out io.Writer, level string) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{Level: lvl}))
}

// WithLogger stores l on ctx so downstream layers can fetch it by key.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger attached to ctx, or slog.Default if none.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
