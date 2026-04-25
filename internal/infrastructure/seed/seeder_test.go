package seed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/seed"
)

type fakeInserter struct {
	count       int
	countErr    error
	insertedAll [][]animal.Animal
	insertErr   error
}

func (f *fakeInserter) Count(ctx context.Context) (int, error) { return f.count, f.countErr }
func (f *fakeInserter) Insert(ctx context.Context, animals []animal.Animal) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.insertedAll = append(f.insertedAll, animals)
	return nil
}

func TestSeed_InsertsRosterWhenTableEmpty(t *testing.T) {
	t.Parallel()

	repo := &fakeInserter{count: 0}
	require.NoError(t, seed.Seed(context.Background(), repo))

	require.Len(t, repo.insertedAll, 1, "insert should fire exactly once")
	require.Len(t, repo.insertedAll[0], 100, "embedded roster has 100 animals")
}

func TestSeed_IsNoOpWhenTableNonEmpty(t *testing.T) {
	t.Parallel()

	repo := &fakeInserter{count: 100}
	require.NoError(t, seed.Seed(context.Background(), repo))
	require.Empty(t, repo.insertedAll, "insert must be skipped when count > 0")
}

func TestSeed_WrapsCountError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	repo := &fakeInserter{countErr: sentinel}
	err := seed.Seed(context.Background(), repo)
	require.Error(t, err)
	require.True(t, errors.Is(err, sentinel))
}

func TestSeed_WrapsInsertError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("db down")
	repo := &fakeInserter{count: 0, insertErr: sentinel}
	err := seed.Seed(context.Background(), repo)
	require.Error(t, err)
	require.True(t, errors.Is(err, sentinel))
}

func TestRoster_ParsesEmbeddedFile(t *testing.T) {
	t.Parallel()

	animals, err := seed.Roster()
	require.NoError(t, err)
	require.Len(t, animals, 100)

	// Spot check: first entry from docs/animals.json.
	require.Equal(t, "roster-000", animals[0].ID, "seeder assigns deterministic roster-index IDs")
	require.Equal(t, "roster-099", animals[99].ID)
	require.Equal(t, "snake", animals[0].Type)
	require.Equal(t, 10, animals[0].Popularity)
	require.Equal(t, []string{"giraffe", "tiger"}, animals[0].Incompatible)
	require.Equal(t, 5, animals[0].MaximumFriends)
}
