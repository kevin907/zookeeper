// Package router wires the chi router, versioned sub-routers, and middleware stack.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/handler"
	zoomw "github.com/kevintrivedi/zoo-api/internal/interfaces/http/middleware"
)

// New constructs the HTTP handler for the service.
//
// Middleware stack (global): RequestID → RealIP → Recoverer → Logger.
// Timeout is mounted on the /api/v1 sub-router so future endpoints can
// register on their own sub-router with a different budget.
func New(
	zoo *handler.ZooHandler,
	pinger handler.Pinger,
	logger *slog.Logger,
	requestTimeout time.Duration,
	readyzTimeout time.Duration,
) http.Handler {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(zoomw.Recoverer)
	r.Use(zoomw.RequestLogger(logger))

	r.Get("/healthz", handler.Liveness)
	r.Get("/readyz", handler.Readiness(pinger, readyzTimeout))

	r.Route("/api/v1", func(sr chi.Router) {
		sr.Use(chimw.Timeout(requestTimeout))
		sr.Get("/zoos/{enclosures}", zoo.Get)
	})
	return r
}
