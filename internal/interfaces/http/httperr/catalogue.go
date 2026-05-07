// Package httperr is the single error catalogue and problem+json renderer
// used by every HTTP handler.
package httperr

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
)

// RFC 7807 type URIs. Every problem+json the service emits uses one of these.
const (
	TypeValidation  = "https://zoo.example/errors/validation"
	TypeNotFound    = "https://zoo.example/errors/not-found"
	TypeInfeasible  = "https://zoo.example/errors/infeasible-assignment"
	TypeUnavailable = "https://zoo.example/errors/dependency-unavailable"
	TypeInternal    = "about:blank"
)

// Problem is the RFC 7807 application/problem+json envelope.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Map converts a layer-wrapped error into a Problem. It is the only place in
// the codebase that decides HTTP status codes for domain errors; handlers
// forward every error through Map.
func Map(err error) Problem {
	switch {
	case errors.Is(err, appzoo.ErrEnclosuresOutOfRange),
		errors.Is(err, appzoo.ErrInvalidEnclosuresInput):
		return Problem{
			Type:   TypeValidation,
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		}
	case errors.Is(err, appzoo.ErrInfeasibleAssignment):
		return Problem{
			Type:   TypeInfeasible,
			Title:  "Unprocessable Entity",
			Status: http.StatusUnprocessableEntity,
			Detail: err.Error(),
		}
	case errors.Is(err, appzoo.ErrDependencyUnavailable):
		return Problem{
			Type:   TypeUnavailable,
			Title:  "Service Unavailable",
			Status: http.StatusServiceUnavailable,
			Detail: "dependency unreachable",
		}
	default:
		return Problem{
			Type:   TypeInternal,
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: "internal error",
		}
	}
}

// Write encodes p as application/problem+json with the appropriate status.
func Write(w http.ResponseWriter, p Problem) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		slog.Error("httperr: encode problem", "error", err)
	}
}
