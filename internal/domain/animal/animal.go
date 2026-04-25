// Package animal declares the Animal value object and its invariants.
package animal

import (
	"errors"
	"fmt"
	"strings"
)

// Animal is an immutable value object describing a single resident.
// ID is a stable identity assigned by the repository (DB primary key or
// roster-index-derived string) that distinguishes otherwise-identical
// individuals; Incompatible holds the set of animal types this animal
// cannot cohabit with; MaximumFriends is the largest number of other
// residents it tolerates in its enclosure.
type Animal struct {
	ID             string
	Type           string
	Popularity     int
	Incompatible   []string
	MaximumFriends int
}

// ErrInvalidAnimal wraps every constructor validation failure.
var ErrInvalidAnimal = errors.New("invalid animal")

// ErrIncompatible is returned by domain callers when two animals cannot cohabit.
var ErrIncompatible = errors.New("animals are incompatible")

// New constructs an Animal after enforcing value-object invariants.
func New(id, typ string, popularity, maxFriends int, incompatible []string) (Animal, error) {
	if strings.TrimSpace(id) == "" {
		return Animal{}, fmt.Errorf("%w: id must not be empty", ErrInvalidAnimal)
	}
	if strings.TrimSpace(typ) == "" {
		return Animal{}, fmt.Errorf("%w: type must not be empty", ErrInvalidAnimal)
	}
	if popularity < 0 {
		return Animal{}, fmt.Errorf("%w: popularity must not be negative", ErrInvalidAnimal)
	}
	if maxFriends < 1 {
		return Animal{}, fmt.Errorf("%w: maximumFriends must be >= 1", ErrInvalidAnimal)
	}

	var copied []string
	if len(incompatible) > 0 {
		copied = make([]string, len(incompatible))
		copy(copied, incompatible)
	}

	return Animal{
		ID:             id,
		Type:           typ,
		Popularity:     popularity,
		MaximumFriends: maxFriends,
		Incompatible:   copied,
	}, nil
}

// IsIncompatibleWith reports whether this animal and other cannot share an
// enclosure. The relation is symmetric: if either side lists the other type,
// they cannot cohabit.
func (a Animal) IsIncompatibleWith(other Animal) bool {
	for _, t := range a.Incompatible {
		if t == other.Type {
			return true
		}
	}
	for _, t := range other.Incompatible {
		if t == a.Type {
			return true
		}
	}
	return false
}
