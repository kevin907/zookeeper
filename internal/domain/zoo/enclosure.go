// Package zoo declares the Enclosure aggregate and its assignment invariants.
package zoo

import (
	"errors"
	"fmt"

	"github.com/kevin907/zookeeper/internal/domain/animal"
)

// Domain errors returned by the Enclosure aggregate.
var (
	ErrCapacityExceeded     = errors.New("enclosure capacity exceeded")
	ErrIncompatibleResident = errors.New("animal is incompatible with an existing resident")
	ErrDuplicateAnimal      = errors.New("animal is already a resident")
)

// Enclosure is the aggregate root for a group of cohabiting animals.
// The zero value is a valid empty enclosure.
type Enclosure struct {
	residents []animal.Animal
}

// Residents returns a defensive copy of the enclosure's current residents.
func (e Enclosure) Residents() []animal.Animal {
	if len(e.residents) == 0 {
		return nil
	}
	out := make([]animal.Animal, len(e.residents))
	copy(out, e.residents)
	return out
}

// IsEmpty reports whether the enclosure has no residents.
func (e Enclosure) IsEmpty() bool {
	return len(e.residents) == 0
}

// CanAccept returns nil if the candidate can join the enclosure without
// violating invariants, or the typed error that blocks it.
// It is a read-only check.
func (e *Enclosure) CanAccept(candidate animal.Animal) error {
	if e.hasResident(candidate) {
		return fmt.Errorf("%w: %s", ErrDuplicateAnimal, candidate.Type)
	}

	// After admission each existing resident (and the candidate) would have
	// len(residents) friends. Reject if any can't tolerate that count.
	friendsAfter := len(e.residents)
	if friendsAfter > candidate.MaximumFriends {
		return fmt.Errorf("%w: %s tolerates at most %d friends",
			ErrCapacityExceeded, candidate.Type, candidate.MaximumFriends)
	}
	for _, r := range e.residents {
		if friendsAfter > r.MaximumFriends {
			return fmt.Errorf("%w: %s tolerates at most %d friends",
				ErrCapacityExceeded, r.Type, r.MaximumFriends)
		}
		if r.IsIncompatibleWith(candidate) {
			return fmt.Errorf("%w: %s vs %s",
				ErrIncompatibleResident, r.Type, candidate.Type)
		}
	}
	return nil
}

// Add enforces every invariant, then admits the animal if all pass.
func (e *Enclosure) Add(candidate animal.Animal) error {
	if err := e.CanAccept(candidate); err != nil {
		return err
	}
	e.residents = append(e.residents, candidate)
	return nil
}

func (e *Enclosure) hasResident(candidate animal.Animal) bool {
	for _, r := range e.residents {
		if animalsEqual(r, candidate) {
			return true
		}
	}
	return false
}

// animalsEqual compares by stable ID only. Two Animals with identical
// field values but distinct IDs are distinct individuals (e.g. two cloned
// tigers from the roster) and may cohabit an enclosure.
func animalsEqual(a, b animal.Animal) bool {
	return a.ID != "" && a.ID == b.ID
}
