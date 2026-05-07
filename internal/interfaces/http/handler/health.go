package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/interfaces/http/httperr"
)

// Pinger is the dependency the readiness handler needs: something that can
// confirm an external resource is reachable within a bounded timeout.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Liveness answers /healthz with a cheap 200.
func Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Readiness answers /readyz by pinging the Pinger under a short timeout.
// On failure the error is routed through httperr.Map so the problem+json
// envelope is built in one place.
func Readiness(p Pinger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		if err := p.Ping(ctx); err != nil {
			httperr.Write(w, httperr.Map(fmt.Errorf("readyz: %w", appzoo.ErrDependencyUnavailable)))
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}
}
