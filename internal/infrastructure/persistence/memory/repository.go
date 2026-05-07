// Package memory is an in-memory Repository adapter used as a test fake and
// as the runtime fallback when POSTGRES_URL is empty.
package memory

import (
	"context"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/domain/animal"
)

// Compile-time assertion that Repository satisfies the application port.
var _ appzoo.Repository = (*Repository)(nil)

// Repository is an immutable slice of animals served from memory.
type Repository struct {
	animals []animal.Animal
}

// New constructs a Repository from a roster, taking a defensive copy so the
// caller can't mutate state after construction.
func New(animals []animal.Animal) *Repository {
	copied := make([]animal.Animal, len(animals))
	copy(copied, animals)
	return &Repository{animals: copied}
}

// List returns a defensive copy of the stored roster. It honours ctx
// cancellation: a cancelled context yields ctx.Err() without a copy.
func (r *Repository) List(ctx context.Context) ([]animal.Animal, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]animal.Animal, len(r.animals))
	copy(out, r.animals)
	return out, nil
}
