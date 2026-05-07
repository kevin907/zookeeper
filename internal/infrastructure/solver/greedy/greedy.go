// Package greedy implements the default popularity-first greedy Solver.
package greedy

import (
	"context"
	"errors"
	"fmt"
	"sort"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/domain/animal"
	domzoo "github.com/kevin907/zookeeper/internal/domain/zoo"
)

// Compile-time assertion that Solver satisfies the application port.
var _ appzoo.Solver = (*Solver)(nil)

// Solver assigns animals greedily: seeds are picked by
// (MaximumFriends DESC, Popularity DESC) so high-capacity animals anchor
// each enclosure, then remaining animals are placed by popularity desc in
// the first enclosure that accepts them. Animals that fit nowhere are
// surfaced in Assignment.Unplaced with a reason code.
type Solver struct{}

// New returns a zero-configuration greedy Solver.
func New() *Solver { return &Solver{} }

// Assign produces a deterministic assignment. Ties break by original roster
// index via stable sort so the output is reproducible for a given input.
func (Solver) Assign(ctx context.Context, animals []animal.Animal, enclosures int) (appzoo.Assignment, error) {
	if err := ctx.Err(); err != nil {
		return appzoo.Assignment{}, err
	}
	if enclosures < 1 {
		return appzoo.Assignment{}, fmt.Errorf("%w: enclosures must be >= 1", appzoo.ErrInfeasibleAssignment)
	}
	if len(animals) < enclosures {
		return appzoo.Assignment{}, fmt.Errorf("%w: fewer animals (%d) than enclosures (%d)",
			appzoo.ErrInfeasibleAssignment, len(animals), enclosures)
	}

	// Two orderings from a single copy: one for seeding (capacity-first) and
	// one for placement (popularity-first). Both are stable over the original
	// roster index, so ties break deterministically.
	indices := seedOrder(animals)
	seedIdx := make(map[int]struct{}, enclosures)
	for i := 0; i < enclosures; i++ {
		seedIdx[indices[i]] = struct{}{}
	}

	result := make([]domzoo.Enclosure, enclosures)
	for slot := 0; slot < enclosures; slot++ {
		a := animals[indices[slot]]
		if err := result[slot].Add(a); err != nil {
			return appzoo.Assignment{}, fmt.Errorf("%w: seed enclosure %d with %q: %w",
				appzoo.ErrInfeasibleAssignment, slot, a.Type, err)
		}
	}

	unplaced := placeRemaining(animals, placementOrder(animals, seedIdx), result)
	return appzoo.Assignment{Enclosures: result, Unplaced: unplaced}, nil
}

// placeRemaining walks the placement order, slots each animal into the first
// accepting enclosure, and records the rest in an Unplaced list with reason.
func placeRemaining(all []animal.Animal, order []int, result []domzoo.Enclosure) []appzoo.UnplacedAnimal {
	var unplaced []appzoo.UnplacedAnimal
	for _, idx := range order {
		a := all[idx]
		placed := false
		var lastErr error
		for i := range result {
			if err := result[i].Add(a); err == nil {
				placed = true
				break
			} else {
				lastErr = err
			}
		}
		if !placed {
			unplaced = append(unplaced, appzoo.UnplacedAnimal{
				Animal: a,
				Reason: classifyReason(lastErr),
			})
		}
	}
	return unplaced
}

// seedOrder returns indices into `animals` ordered by
// (MaximumFriends DESC, Popularity DESC, original index ASC). High-capacity
// seeds anchor each enclosure so mf=1 animals don't strangle a slot
// (QA cases 2, 4, 7).
func seedOrder(animals []animal.Animal) []int {
	idx := make([]int, len(animals))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(i, j int) bool {
		a, b := animals[idx[i]], animals[idx[j]]
		if a.MaximumFriends != b.MaximumFriends {
			return a.MaximumFriends > b.MaximumFriends
		}
		return a.Popularity > b.Popularity
	})
	return idx
}

// placementOrder returns indices into `animals` excluding the seed set,
// ordered by Popularity DESC for a popularity-first greedy fill.
func placementOrder(animals []animal.Animal, seed map[int]struct{}) []int {
	out := make([]int, 0, len(animals)-len(seed))
	for i := range animals {
		if _, isSeed := seed[i]; isSeed {
			continue
		}
		out = append(out, i)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return animals[out[i]].Popularity > animals[out[j]].Popularity
	})
	return out
}

// classifyReason maps a domain-level rejection error to the short string
// codes callers branch on (QA cases 1, 3, 6 surfacing).
func classifyReason(err error) string {
	switch {
	case errors.Is(err, domzoo.ErrCapacityExceeded):
		return appzoo.ReasonCapacity
	case errors.Is(err, domzoo.ErrIncompatibleResident):
		return appzoo.ReasonIncompatible
	case errors.Is(err, domzoo.ErrDuplicateAnimal):
		return appzoo.ReasonDuplicate
	default:
		return appzoo.ReasonUnknown
	}
}
