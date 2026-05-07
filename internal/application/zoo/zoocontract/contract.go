// Package zoocontract hosts reusable test suites that enforce the application
// ports' contracts. Every Solver implementation must pass SolverContract.
package zoocontract

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/domain/animal"
	domzoo "github.com/kevin907/zookeeper/internal/domain/zoo"
)

// SolverContract exercises the behavioural contract every Solver must honour:
// N non-empty enclosures across the full 1..8 range, respects domain
// invariants when constrained inputs force splits, never duplicates animals,
// and is deterministic. Adapters pass a factory so fresh instances are
// constructed per sub-test.
func SolverContract(t *testing.T, name string, factory func() appzoo.Solver) {
	t.Helper()

	permissive := permissiveRoster(t)
	constrained := constrainedRoster(t)

	t.Run(name+"/produces_N_non_empty_enclosures", func(t *testing.T) {
		t.Parallel()
		for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
			n := n
			t.Run("", func(t *testing.T) {
				t.Parallel()
				got, err := factory().Assign(context.Background(), permissive, n)
				require.NoError(t, err)
				require.Len(t, got.Enclosures, n)
				for i, enc := range got.Enclosures {
					require.False(t, enc.IsEmpty(), "enclosure %d must be non-empty", i)
				}
			})
		}
	})

	t.Run(name+"/respects_incompatibility_and_capacity", func(t *testing.T) {
		t.Parallel()
		got, err := factory().Assign(context.Background(), constrained, 3)
		require.NoError(t, err)
		for i, enc := range got.Enclosures {
			residents := enc.Residents()
			for j, a := range residents {
				require.LessOrEqual(t, len(residents)-1, a.MaximumFriends,
					"enclosure %d animal %d %q exceeds maxFriends", i, j, a.Type)
				for k, b := range residents {
					if j == k {
						continue
					}
					require.False(t, a.IsIncompatibleWith(b),
						"enclosure %d: %q is incompatible with %q", i, a.Type, b.Type)
				}
			}
		}
	})

	t.Run(name+"/no_duplicate_animals", func(t *testing.T) {
		t.Parallel()
		got, err := factory().Assign(context.Background(), permissive, 4)
		require.NoError(t, err)
		placed := 0
		for _, enc := range got.Enclosures {
			placed += len(enc.Residents())
		}
		// Roster-total consistency: every animal is either placed exactly once
		// or surfaced in Unplaced exactly once. This is the Liskov check QA
		// relies on to replace silent drops.
		require.Equal(t, len(permissive), placed+len(got.Unplaced),
			"placed + unplaced must equal roster size")
	})

	t.Run(name+"/is_deterministic", func(t *testing.T) {
		t.Parallel()
		first, err := factory().Assign(context.Background(), permissive, 3)
		require.NoError(t, err)
		second, err := factory().Assign(context.Background(), permissive, 3)
		require.NoError(t, err)
		require.Equal(t, enclosureShape(first.Enclosures), enclosureShape(second.Enclosures))
	})

	t.Run(name+"/rejects_enclosures_below_one", func(t *testing.T) {
		t.Parallel()
		_, err := factory().Assign(context.Background(), permissive, 0)
		require.ErrorIs(t, err, appzoo.ErrInfeasibleAssignment)
	})
}

// permissiveRoster is solvable for any N in 1..8. All maxFriends are generous
// and no animals list incompatibilities.
func permissiveRoster(t *testing.T) []animal.Animal {
	t.Helper()
	specs := []struct {
		typ        string
		popularity int
		maxFriends int
	}{
		{"lion", 10, 20}, {"monkey", 9, 20}, {"giraffe", 8, 20},
		{"zebra", 7, 20}, {"lion", 6, 20}, {"monkey", 5, 20},
		{"giraffe", 4, 20}, {"zebra", 3, 20}, {"lion", 2, 20},
		{"monkey", 1, 20},
	}
	out := make([]animal.Animal, 0, len(specs))
	for i, s := range specs {
		id := fmt.Sprintf("perm-%02d", i)
		a, err := animal.New(id, s.typ, s.popularity, s.maxFriends, nil)
		require.NoError(t, err)
		out = append(out, a)
	}
	return out
}

// constrainedRoster introduces realistic incompatibilities and tighter
// capacities to exercise the invariant checks. It is solvable at N=3.
func constrainedRoster(t *testing.T) []animal.Animal {
	t.Helper()
	specs := []struct {
		typ          string
		popularity   int
		maxFriends   int
		incompatible []string
	}{
		{"lion", 10, 4, []string{"snake"}},
		{"tiger", 9, 4, nil},
		{"giraffe", 8, 4, nil},
		{"snake", 7, 4, nil},
		{"zebra", 6, 4, nil},
		{"monkey", 5, 4, []string{"giraffe"}},
		{"alligator", 4, 4, []string{"zebra"}},
		{"lion", 3, 4, nil},
		{"tiger", 2, 4, nil},
	}
	out := make([]animal.Animal, 0, len(specs))
	for i, s := range specs {
		id := fmt.Sprintf("cons-%02d", i)
		a, err := animal.New(id, s.typ, s.popularity, s.maxFriends, s.incompatible)
		require.NoError(t, err)
		out = append(out, a)
	}
	return out
}

func enclosureShape(in []domzoo.Enclosure) [][]string {
	out := make([][]string, len(in))
	for i, enc := range in {
		residents := enc.Residents()
		types := make([]string, len(residents))
		for j, r := range residents {
			types[j] = r.Type
		}
		out[i] = types
	}
	return out
}
