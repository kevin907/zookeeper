package zoo

import (
	"context"
	"fmt"
)

const (
	minEnclosures = 1
	maxEnclosures = 8
)

// ZooService orchestrates the AssignEnclosures use case by composing a
// Repository (roster) and a Solver (assignment algorithm).
type ZooService struct {
	repo   Repository
	solver Solver
}

// NewZooService wires a Repository and a Solver into a ZooService.
func NewZooService(repo Repository, solver Solver) *ZooService {
	return &ZooService{repo: repo, solver: solver}
}

// AssignEnclosures loads the roster and returns an Assignment covering both
// the placed animals (by enclosure) and any that could not be placed. It
// enforces the [1, 8] contract at the service boundary and wraps adapter
// errors so callers can distinguish layers.
func (s *ZooService) AssignEnclosures(ctx context.Context, enclosures int) (Assignment, error) {
	if enclosures < minEnclosures || enclosures > maxEnclosures {
		return Assignment{}, fmt.Errorf("%w: got %d", ErrEnclosuresOutOfRange, enclosures)
	}

	animals, err := s.repo.List(ctx)
	if err != nil {
		return Assignment{}, fmt.Errorf("load roster: %w", err)
	}

	result, err := s.solver.Assign(ctx, animals, enclosures)
	if err != nil {
		return Assignment{}, fmt.Errorf("solve: %w", err)
	}
	return result, nil
}
