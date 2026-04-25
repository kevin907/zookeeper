package zoo_test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
	"github.com/kevintrivedi/zoo-api/internal/domain/zoo"
)

var idCounter atomic.Uint64

// mustAnimal auto-generates a unique ID per call so tests that want distinct
// individuals get them without plumbing IDs through every case. Tests that
// need to assert duplicate-individual rejection reuse the returned Animal.
func mustAnimal(t *testing.T, typ string, popularity, maxFriends int, incompatible []string) animal.Animal {
	t.Helper()
	id := fmt.Sprintf("test-%d", idCounter.Add(1))
	a, err := animal.New(id, typ, popularity, maxFriends, incompatible)
	require.NoError(t, err)
	return a
}

func TestEnclosure_EmptyByDefault(t *testing.T) {
	t.Parallel()

	var e zoo.Enclosure
	require.True(t, e.IsEmpty())
	require.Empty(t, e.Residents())
}

func TestEnclosure_AddAppendsResident(t *testing.T) {
	t.Parallel()

	var e zoo.Enclosure
	lion := mustAnimal(t, "lion", 5, 3, nil)
	require.NoError(t, e.Add(lion))
	require.False(t, e.IsEmpty())
	require.Len(t, e.Residents(), 1)
}

func TestEnclosure_Add_RejectsDuplicateSameIndividual(t *testing.T) {
	t.Parallel()

	var e zoo.Enclosure
	lion := mustAnimal(t, "lion", 5, 3, nil)
	require.NoError(t, e.Add(lion))
	err := e.Add(lion)
	require.ErrorIs(t, err, zoo.ErrDuplicateAnimal,
		"adding the same Animal (same ID) twice must be rejected")
}

func TestEnclosure_Add_AllowsTwinsWithDistinctIDs(t *testing.T) {
	t.Parallel()

	// Two tigers with identical field values but distinct IDs are distinct
	// individuals and must be allowed to cohabit (QA Case 5).
	twinA, err := animal.New("tiger-a", "tiger", 8, 3, []string{"snake"})
	require.NoError(t, err)
	twinB, err := animal.New("tiger-b", "tiger", 8, 3, []string{"snake"})
	require.NoError(t, err)

	var e zoo.Enclosure
	require.NoError(t, e.Add(twinA))
	require.NoError(t, e.Add(twinB), "twins with distinct IDs cohabit")
	require.Len(t, e.Residents(), 2)
}

func TestEnclosure_Add_RejectsIncompatibleResident(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b animal.Animal
	}{
		{
			name: "first animal lists second as incompatible",
			a:    mustAnimal(t, "snake", 10, 5, []string{"giraffe"}),
			b:    mustAnimal(t, "giraffe", 5, 4, nil),
		},
		{
			name: "second animal lists first as incompatible",
			a:    mustAnimal(t, "lion", 5, 3, nil),
			b:    mustAnimal(t, "alligator", 8, 4, []string{"lion"}),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var e zoo.Enclosure
			require.NoError(t, e.Add(tc.a))
			err := e.Add(tc.b)
			require.ErrorIs(t, err, zoo.ErrIncompatibleResident)
		})
	}
}

func TestEnclosure_Add_RejectsWhenCapacityExceeded(t *testing.T) {
	t.Parallel()

	// candidate.MaximumFriends = 1 — can tolerate only one other friend.
	solo1 := mustAnimal(t, "lion", 1, 1, nil)
	solo2 := mustAnimal(t, "lion", 2, 1, nil)
	solo3 := mustAnimal(t, "lion", 3, 1, nil)

	var e zoo.Enclosure
	require.NoError(t, e.Add(solo1))
	require.NoError(t, e.Add(solo2))
	err := e.Add(solo3)
	require.ErrorIs(t, err, zoo.ErrCapacityExceeded)
}

func TestEnclosure_Add_CapacityLimitedByMostRestrictiveResident(t *testing.T) {
	t.Parallel()

	// The most restrictive resident (maxFriends=1) caps the enclosure at 2 total
	// even though the others can tolerate more.
	restrictive := mustAnimal(t, "alligator", 8, 1, nil)
	tolerant := mustAnimal(t, "monkey", 5, 10, nil)
	candidate := mustAnimal(t, "giraffe", 6, 10, nil)

	var e zoo.Enclosure
	require.NoError(t, e.Add(restrictive))
	require.NoError(t, e.Add(tolerant))
	err := e.Add(candidate)
	require.ErrorIs(t, err, zoo.ErrCapacityExceeded)
}

func TestEnclosure_CanAccept_DoesNotMutate(t *testing.T) {
	t.Parallel()

	var e zoo.Enclosure
	lion := mustAnimal(t, "lion", 5, 3, nil)
	require.NoError(t, e.Add(lion))

	snake := mustAnimal(t, "snake", 10, 5, nil)
	require.NoError(t, e.CanAccept(snake))
	require.Len(t, e.Residents(), 1, "CanAccept must be a read-only check")
}

func TestEnclosure_Residents_ReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	var e zoo.Enclosure
	lion := mustAnimal(t, "lion", 5, 3, nil)
	require.NoError(t, e.Add(lion))

	got := e.Residents()
	got[0] = mustAnimal(t, "tiger", 9, 4, nil)

	require.Equal(t, "lion", e.Residents()[0].Type, "mutating returned slice must not affect aggregate")
}
