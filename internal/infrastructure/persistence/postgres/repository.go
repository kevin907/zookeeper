package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
)

// Compile-time assertion that Repository satisfies the application port.
var _ appzoo.Repository = (*Repository)(nil)

// Repository is a Postgres-backed implementation of application/zoo.Repository.
// pgx types do not leak past this package; callers see only domain types.
type Repository struct {
	pool *pgxpool.Pool
}

// New wraps the given pool as a Repository.
func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// List loads all animals from the JSONB data column in insertion order.
// Each returned Animal carries the DB primary key as its stable ID.
func (r *Repository) List(ctx context.Context) ([]animal.Animal, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, data FROM animals ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("query animals: %w", err)
	}
	defer rows.Close()

	var out []animal.Animal
	for rows.Next() {
		var id int64
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, fmt.Errorf("scan animal row: %w", err)
		}
		a, err := decodeRow(id, raw)
		if err != nil {
			return nil, fmt.Errorf("decode animal row: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate animal rows: %w", err)
	}
	return out, nil
}

// Count returns the number of rows currently in the animals table. Used by
// the seeder to stay idempotent.
func (r *Repository) Count(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM animals`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count animals: %w", err)
	}
	return n, nil
}

// Insert batch-inserts animals into the JSONB-backed table. Used by the seeder.
func (r *Repository) Insert(ctx context.Context, animals []animal.Animal) error {
	if len(animals) == 0 {
		return nil
	}
	rows := make([][]any, 0, len(animals))
	for _, a := range animals {
		payload, err := json.Marshal(toRow(a))
		if err != nil {
			return fmt.Errorf("encode animal %q: %w", a.Type, err)
		}
		rows = append(rows, []any{payload})
	}
	if _, err := r.pool.CopyFrom(ctx,
		[]string{"animals"},
		[]string{"data"},
		&copyFromRows{rows: rows},
	); err != nil {
		return fmt.Errorf("copy animals: %w", err)
	}
	return nil
}

// Ping runs a cheap SELECT 1 under the caller's context; used by /readyz.
func (r *Repository) Ping(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `SELECT 1`); err != nil {
		return fmt.Errorf("postgres ping: %w", err)
	}
	return nil
}

// animalRow is the wire shape stored inside the data JSONB column. This is
// the only place in the codebase that carries json struct tags for domain
// fields. It is not a domain type.
type animalRow struct {
	Type           string   `json:"type"`
	Popularity     int      `json:"popularity"`
	Incompatible   []string `json:"incompatible"`
	MaximumFriends int      `json:"maximumFriends"`
}

func decodeRow(id int64, raw []byte) (animal.Animal, error) {
	var r animalRow
	if err := json.Unmarshal(raw, &r); err != nil {
		return animal.Animal{}, fmt.Errorf("unmarshal row: %w", err)
	}
	return animal.New(strconv.FormatInt(id, 10), r.Type, r.Popularity, r.MaximumFriends, r.Incompatible)
}

func toRow(a animal.Animal) animalRow {
	return animalRow{
		Type:           a.Type,
		Popularity:     a.Popularity,
		Incompatible:   a.Incompatible,
		MaximumFriends: a.MaximumFriends,
	}
}

// copyFromRows adapts a pre-built slice of rows to pgx's CopyFromSource.
type copyFromRows struct {
	rows [][]any
	idx  int
}

func (c *copyFromRows) Next() bool {
	if c.idx >= len(c.rows) {
		return false
	}
	c.idx++
	return true
}

func (c *copyFromRows) Values() ([]any, error) { return c.rows[c.idx-1], nil }
func (c *copyFromRows) Err() error             { return nil }
