// Package solver hosts the NewSolver factory that returns a concrete Solver
// by name, keeping the wire-time selection in one place.
package solver

import (
	"fmt"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/solver/greedy"
)

// Default is the solver name used when none is configured.
const Default = "greedy"

// NewSolver returns the Solver registered under the given name.
// An empty name resolves to Default. Unknown names return an error.
func NewSolver(name string) (appzoo.Solver, error) {
	switch name {
	case "", Default:
		return greedy.New(), nil
	default:
		return nil, fmt.Errorf("unknown solver %q", name)
	}
}
