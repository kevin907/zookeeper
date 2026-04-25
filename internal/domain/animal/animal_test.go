package animal_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
)

func TestNew_ValidatesInputs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		id         string
		typ        string
		popularity int
		maxFriends int
		wantErr    error
	}{
		{"happy path", "lion-1", "lion", 5, 3, nil},
		{"empty id", "", "lion", 1, 1, animal.ErrInvalidAnimal},
		{"whitespace id", "   ", "lion", 1, 1, animal.ErrInvalidAnimal},
		{"empty type", "lion-1", "", 1, 1, animal.ErrInvalidAnimal},
		{"whitespace type", "lion-1", "   ", 1, 1, animal.ErrInvalidAnimal},
		{"negative popularity", "lion-1", "lion", -1, 1, animal.ErrInvalidAnimal},
		{"zero max friends", "lion-1", "lion", 1, 0, animal.ErrInvalidAnimal},
		{"negative max friends", "lion-1", "lion", 1, -1, animal.ErrInvalidAnimal},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := animal.New(tc.id, tc.typ, tc.popularity, tc.maxFriends, nil)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.id, got.ID)
			require.Equal(t, tc.typ, got.Type)
			require.Equal(t, tc.popularity, got.Popularity)
			require.Equal(t, tc.maxFriends, got.MaximumFriends)
		})
	}
}

func TestNew_DefensivelyCopiesIncompatible(t *testing.T) {
	t.Parallel()

	incompat := []string{"zebra", "tiger"}
	a, err := animal.New("snake-1", "snake", 10, 5, incompat)
	require.NoError(t, err)

	incompat[0] = "mutated"
	require.Equal(t, "zebra", a.Incompatible[0], "constructor must copy the slice to preserve immutability")
}

func TestIsIncompatibleWith_Symmetric(t *testing.T) {
	t.Parallel()

	snake, err := animal.New("snake-1", "snake", 10, 5, []string{"giraffe", "tiger"})
	require.NoError(t, err)
	giraffe, err := animal.New("giraffe-1", "giraffe", 10, 4, nil)
	require.NoError(t, err)
	lion, err := animal.New("lion-1", "lion", 5, 3, nil)
	require.NoError(t, err)

	require.True(t, snake.IsIncompatibleWith(giraffe), "snake lists giraffe")
	require.True(t, giraffe.IsIncompatibleWith(snake), "relation must be symmetric")
	require.False(t, snake.IsIncompatibleWith(lion), "no listing in either direction")
	require.False(t, lion.IsIncompatibleWith(snake), "no listing in either direction")
}

func TestIsIncompatibleWith_ReversePairing(t *testing.T) {
	t.Parallel()

	// If only the *other* side lists incompatibility, both sides must still
	// return true — the relation is symmetric regardless of which animal's
	// Incompatible list records it.
	alligator, err := animal.New("alligator-1", "alligator", 8, 2, []string{"lion"})
	require.NoError(t, err)
	lion, err := animal.New("lion-1", "lion", 5, 3, nil)
	require.NoError(t, err)

	require.True(t, lion.IsIncompatibleWith(alligator))
	require.True(t, alligator.IsIncompatibleWith(lion))
}

func TestErrInvalidAnimal_WrapsConstructorFailures(t *testing.T) {
	t.Parallel()

	_, err := animal.New("", "lion", 1, 1, nil)
	require.Error(t, err)
	require.True(t, errors.Is(err, animal.ErrInvalidAnimal))
}
