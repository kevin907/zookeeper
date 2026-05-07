// Package handler hosts thin HTTP handlers that parse requests, call services,
// and render responses through the dto package.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/interfaces/http/dto"
	"github.com/kevin907/zookeeper/internal/interfaces/http/httperr"
)

// ZooAssigner is the service-shaped dependency the handler consumes.
// Defined here (consumer-owned) so the handler doesn't depend on a concrete
// service type.
type ZooAssigner interface {
	AssignEnclosures(ctx context.Context, enclosures int) (appzoo.Assignment, error)
}

// ZooHandler serves GET /api/v1/zoos/{enclosures}.
type ZooHandler struct {
	svc ZooAssigner
}

// NewZooHandler wires a ZooAssigner into a handler.
func NewZooHandler(svc ZooAssigner) *ZooHandler { return &ZooHandler{svc: svc} }

// Get parses the path parameter, calls the service, and renders the response
// as JSON (or problem+json on error).
func (h *ZooHandler) Get(w http.ResponseWriter, r *http.Request) {
	n, err := parseEnclosures(chi.URLParam(r, "enclosures"))
	if err != nil {
		httperr.Write(w, httperr.Map(err))
		return
	}
	assignment, err := h.svc.AssignEnclosures(r.Context(), n)
	if err != nil {
		httperr.Write(w, httperr.Map(err))
		return
	}
	writeJSON(w, http.StatusOK, dto.ZooFromAssignment(assignment))
}

func parseEnclosures(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: %q is not an integer", appzoo.ErrInvalidEnclosuresInput, raw)
	}
	return n, nil
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
