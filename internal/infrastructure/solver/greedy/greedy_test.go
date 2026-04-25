package greedy_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/application/zoo/zoocontract"
	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/solver/greedy"
)

var greedyIDCounter atomic.Uint64

// TestGreedy_ContractSuite wires the greedy solver into the reusable
// application-layer contract test. Any future solver must pass the same suite.
func TestGreedy_ContractSuite(t *testing.T) {
	zoocontract.SolverContract(t, "greedy", func() appzoo.Solver { return greedy.New() })
}

func mustAnimal(t *testing.T, typ string, popularity, maxFriends int) animal.Animal {
	t.Helper()
	id := fmt.Sprintf("gr-%d", greedyIDCounter.Add(1))
	a, err := animal.New(id, typ, popularity, maxFriends, nil)
	require.NoError(t, err)
	return a
}

func TestGreedy_SingleEnclosure(t *testing.T) {
	t.Parallel()

	roster := []animal.Animal{
		mustAnimal(t, "lion", 5, 10),
		mustAnimal(t, "monkey", 3, 10),
	}
	got, err := greedy.New().Assign(context.Background(), roster, 1)
	require.NoError(t, err)
	require.Len(t, got.Enclosures, 1)
	require.Len(t, got.Enclosures[0].Residents(), 2)
	require.Empty(t, got.Unplaced)
}

func TestGreedy_ReturnsInfeasibleWhenFewerAnimalsThanEnclosures(t *testing.T) {
	t.Parallel()

	roster := []animal.Animal{mustAnimal(t, "lion", 5, 3)}
	_, err := greedy.New().Assign(context.Background(), roster, 3)
	require.ErrorIs(t, err, appzoo.ErrInfeasibleAssignment)
}

func TestGreedy_RespectsCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := greedy.New().Assign(ctx, []animal.Animal{mustAnimal(t, "lion", 5, 3)}, 1)
	require.ErrorIs(t, err, context.Canceled)
}

// TestGreedy_PrefersHighCapacitySeeds asserts QA cases 4 and 7: when the
// roster mixes a top-popularity mf=1 animal with lower-popularity but
// higher-capacity animals, the seed slot goes to the high-capacity one so
// the enclosure doesn't stall at size 2.
func TestGreedy_PrefersHighCapacitySeeds(t *testing.T) {
	t.Parallel()

	roster := []animal.Animal{
		mustAnimal(t, "snake", 10, 1),  // highest popularity but solo
		mustAnimal(t, "giraffe", 9, 5), // lower popularity, high capacity
		mustAnimal(t, "lion", 8, 5),
		mustAnimal(t, "monkey", 7, 5),
		mustAnimal(t, "tiger", 6, 5),
	}
	got, err := greedy.New().Assign(context.Background(), roster, 1)
	require.NoError(t, err)
	require.Len(t, got.Enclosures, 1)

	residents := got.Enclosures[0].Residents()
	require.NotEmpty(t, residents)
	require.Equal(t, "giraffe", residents[0].Type,
		"seed slot should go to the highest-capacity animal, not the mf=1 snake")
}

// TestGreedy_SurfacesUnplacedWithReason asserts QA cases 1, 3, 6: when an
// animal cannot fit in any enclosure, it appears in Unplaced with a
// machine-readable reason code.
func TestGreedy_SurfacesUnplacedWithReason(t *testing.T) {
	t.Parallel()

	// Two enclosures seeded with mf=5 animals of different types. A third
	// animal has incompat=[both types], so it cannot fit in either enclosure.
	roster := []animal.Animal{
		mustAnimalWithIncompat(t, "giraffe", 10, 5, nil),
		mustAnimalWithIncompat(t, "zebra", 9, 5, nil),
		mustAnimalWithIncompat(t, "snake", 5, 5, []string{"giraffe", "zebra"}),
	}
	got, err := greedy.New().Assign(context.Background(), roster, 2)
	require.NoError(t, err)

	require.Len(t, got.Unplaced, 1)
	require.Equal(t, "snake", got.Unplaced[0].Animal.Type)
	require.Equal(t, appzoo.ReasonIncompatible, got.Unplaced[0].Reason)
}

func mustAnimalWithIncompat(t *testing.T, typ string, popularity, maxFriends int, incompat []string) animal.Animal {
	t.Helper()
	id := fmt.Sprintf("gri-%d", greedyIDCounter.Add(1))
	a, err := animal.New(id, typ, popularity, maxFriends, incompat)
	require.NoError(t, err)
	return a
}
