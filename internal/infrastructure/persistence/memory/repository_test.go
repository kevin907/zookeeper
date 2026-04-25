package memory_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/persistence/memory"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/solver/greedy"
)

var memIDCounter atomic.Uint64

func mustAnimal(t *testing.T, typ string) animal.Animal {
	t.Helper()
	id := fmt.Sprintf("mem-%d", memIDCounter.Add(1))
	a, err := animal.New(id, typ, 1, 1, nil)
	require.NoError(t, err)
	return a
}

func TestRepository_List(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   []animal.Animal
		wantLen int
	}{
		{"empty roster", nil, 0},
		{"populated roster", []animal.Animal{mustAnimal(t, "lion"), mustAnimal(t, "zebra")}, 2},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := memory.New(tc.input)
			got, err := r.List(context.Background())
			require.NoError(t, err)
			require.Len(t, got, tc.wantLen)
		})
	}
}

func TestRepository_List_ReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	lion := mustAnimal(t, "lion")
	r := memory.New([]animal.Animal{lion})

	got, err := r.List(context.Background())
	require.NoError(t, err)
	got[0] = mustAnimal(t, "tiger")

	again, err := r.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "lion", again[0].Type, "mutating returned slice must not affect repo")
}

func TestNew_CopiesInputSlice(t *testing.T) {
	t.Parallel()

	input := []animal.Animal{mustAnimal(t, "lion")}
	r := memory.New(input)

	input[0] = mustAnimal(t, "tiger")

	got, err := r.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "lion", got[0].Type, "constructor must copy input to preserve immutability")
}

func TestRepository_List_RespectsContextCancellation(t *testing.T) {
	t.Parallel()

	r := memory.New([]animal.Animal{mustAnimal(t, "lion")})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := r.List(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestRepository_List_ConcurrentReadsAreSafe(t *testing.T) {
	t.Parallel()

	r := memory.New([]animal.Animal{mustAnimal(t, "lion"), mustAnimal(t, "zebra")})

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := r.List(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 2)
		}()
	}
	wg.Wait()
}

// TestRepository_LiskovSubstitution wires the memory Repository into the real
// application service and greedy solver end-to-end. Satisfies Task 04's
// Liskov check — the memory adapter is substitutable for any Repository
// implementation against the service's public contract.
func TestRepository_LiskovSubstitution(t *testing.T) {
	t.Parallel()

	roster := []animal.Animal{
		mustAnimal(t, "lion"), mustAnimal(t, "tiger"),
		mustAnimal(t, "giraffe"), mustAnimal(t, "zebra"),
	}
	svc := appzoo.NewZooService(memory.New(roster), greedy.New())

	out, err := svc.AssignEnclosures(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, out.Enclosures, 2)
	for i, enc := range out.Enclosures {
		require.False(t, enc.IsEmpty(), "enclosure %d must be non-empty", i)
	}

	_, err = svc.AssignEnclosures(context.Background(), 0)
	require.ErrorIs(t, err, appzoo.ErrEnclosuresOutOfRange, "service-level validation still fires")
}
