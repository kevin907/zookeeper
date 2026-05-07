// Package zoo hosts the AssignEnclosures use case and its consumer-owned ports.
package zoo

import (
	"context"

	"github.com/kevin907/zookeeper/internal/domain/animal"
	domzoo "github.com/kevin907/zookeeper/internal/domain/zoo"
)

// Repository returns the fixed roster of animals to be assigned.
// Defined here, not next to its implementations.
type Repository interface {
	List(ctx context.Context) ([]animal.Animal, error)
}

// Solver assigns a slice of animals into the requested number of enclosures
// and reports any animals that could not be placed, so the caller sees an
// explicit signal instead of a silent drop (QA cases 1, 2, 3, 4, 6, 7).
// Implementations must honour domain invariants and be deterministic for a
// given input.
type Solver interface {
	Assign(ctx context.Context, animals []animal.Animal, enclosures int) (Assignment, error)
}

// Assignment is the result of a Solver.Assign call: enclosures plus any
// animals the solver could not accommodate.
type Assignment struct {
	Enclosures []domzoo.Enclosure
	Unplaced   []UnplacedAnimal
}

// UnplacedAnimal pairs an animal with the reason it could not be placed.
// Reason is one of "capacity", "incompatible", "duplicate", or "unknown".
// Callers branch on the code instead of parsing prose.
type UnplacedAnimal struct {
	Animal animal.Animal
	Reason string
}

// Reason codes for UnplacedAnimal.Reason.
const (
	ReasonCapacity     = "capacity"
	ReasonIncompatible = "incompatible"
	ReasonDuplicate    = "duplicate"
	ReasonUnknown      = "unknown"
)
