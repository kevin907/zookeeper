// Package seed holds the embedded animals roster and the idempotent seeder
// invoked during application boot.
package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
)

//go:embed animals.json
var rosterBytes []byte

// Inserter is the consumer-owned port the seeder needs from a repository:
// a way to ask "is this empty?" and a way to batch-insert. The Postgres
// adapter satisfies it; tests substitute a fake.
type Inserter interface {
	Count(ctx context.Context) (int, error)
	Insert(ctx context.Context, animals []animal.Animal) error
}

// Seed loads the embedded roster and inserts it into the repository if the
// destination table is currently empty. Running it a second time is a no-op.
func Seed(ctx context.Context, repo Inserter) error {
	n, err := repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("seed: count rows: %w", err)
	}
	if n > 0 {
		return nil
	}
	animals, err := Roster()
	if err != nil {
		return fmt.Errorf("seed: load roster: %w", err)
	}
	if err := repo.Insert(ctx, animals); err != nil {
		return fmt.Errorf("seed: insert roster: %w", err)
	}
	return nil
}

// Roster parses the embedded animals.json into domain Animals. Exported for
// callers that need the in-memory fallback (cmd/api).
func Roster() ([]animal.Animal, error) {
	var rows []seedRow
	if err := json.Unmarshal(rosterBytes, &rows); err != nil {
		return nil, fmt.Errorf("parse embedded animals.json: %w", err)
	}
	out := make([]animal.Animal, 0, len(rows))
	for i, r := range rows {
		// Memory-fallback IDs are derived from roster position; the Postgres
		// path overwrites this with the DB primary key via repository.List().
		id := fmt.Sprintf("roster-%03d", i)
		a, err := animal.New(id, r.Type, r.Popularity, r.MaximumFriends, r.Incompatible)
		if err != nil {
			return nil, fmt.Errorf("row %d (%s): %w", i, r.Type, err)
		}
		out = append(out, a)
	}
	return out, nil
}

// seedRow mirrors the on-disk JSON shape. It lives in the seed package so the
// postgres adapter doesn't have to share its wire type with a file format.
type seedRow struct {
	Type           string   `json:"type"`
	Popularity     int      `json:"popularity"`
	Incompatible   []string `json:"incompatible"`
	MaximumFriends int      `json:"maximumFriends"`
}
