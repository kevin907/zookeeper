package zoo_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/domain/animal"
	domzoo "github.com/kevin907/zookeeper/internal/domain/zoo"
)

var idCounter atomic.Uint64

type fakeRepo struct {
	animals []animal.Animal
	err     error
	ctxSeen context.Context
	calls   int
}

func (f *fakeRepo) List(ctx context.Context) ([]animal.Animal, error) {
	f.ctxSeen = ctx
	f.calls++
	return f.animals, f.err
}

type fakeSolver struct {
	result      zoo.Assignment
	err         error
	ctxSeen     context.Context
	animalsSeen []animal.Animal
	countSeen   int
}

func (f *fakeSolver) Assign(ctx context.Context, animals []animal.Animal, enclosures int) (zoo.Assignment, error) {
	f.ctxSeen = ctx
	f.animalsSeen = animals
	f.countSeen = enclosures
	return f.result, f.err
}

func mustAnimal(t *testing.T, typ string) animal.Animal {
	t.Helper()
	id := fmt.Sprintf("svc-%d", idCounter.Add(1))
	a, err := animal.New(id, typ, 1, 1, nil)
	require.NoError(t, err)
	return a
}

func TestAssignEnclosures_ReturnsErrorWhenOutOfRange(t *testing.T) {
	t.Parallel()

	cases := []int{-1, 0, 9, 100}
	for _, n := range cases {
		n := n
		t.Run("", func(t *testing.T) {
			t.Parallel()
			svc := zoo.NewZooService(&fakeRepo{}, &fakeSolver{})
			_, err := svc.AssignEnclosures(context.Background(), n)
			require.ErrorIs(t, err, zoo.ErrEnclosuresOutOfRange)
		})
	}
}

func TestAssignEnclosures_WrapsRepositoryError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{err: io.ErrUnexpectedEOF}
	svc := zoo.NewZooService(repo, &fakeSolver{})
	_, err := svc.AssignEnclosures(context.Background(), 3)

	require.Error(t, err)
	require.Contains(t, err.Error(), "load roster")
	require.True(t, errors.Is(err, io.ErrUnexpectedEOF))
}

func TestAssignEnclosures_WrapsSolverError(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{animals: []animal.Animal{mustAnimal(t, "lion")}}
	solver := &fakeSolver{err: zoo.ErrInfeasibleAssignment}
	svc := zoo.NewZooService(repo, solver)

	_, err := svc.AssignEnclosures(context.Background(), 3)

	require.Error(t, err)
	require.Contains(t, err.Error(), "solve")
	require.True(t, errors.Is(err, zoo.ErrInfeasibleAssignment))
}

func TestAssignEnclosures_ReturnsSolverOutput(t *testing.T) {
	t.Parallel()

	animals := []animal.Animal{mustAnimal(t, "lion"), mustAnimal(t, "zebra")}
	expected := zoo.Assignment{
		Enclosures: []domzoo.Enclosure{{}, {}},
		Unplaced: []zoo.UnplacedAnimal{
			{Animal: mustAnimal(t, "alligator"), Reason: zoo.ReasonIncompatible},
		},
	}

	repo := &fakeRepo{animals: animals}
	solver := &fakeSolver{result: expected}
	svc := zoo.NewZooService(repo, solver)

	got, err := svc.AssignEnclosures(context.Background(), 2)
	require.NoError(t, err)
	require.Len(t, got.Enclosures, len(expected.Enclosures))
	require.Len(t, got.Unplaced, 1, "service forwards Unplaced untouched")
	require.Equal(t, zoo.ReasonIncompatible, got.Unplaced[0].Reason)
	require.Equal(t, animals, solver.animalsSeen, "service forwards the roster as-is")
	require.Equal(t, 2, solver.countSeen)
}

func TestAssignEnclosures_PropagatesContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &fakeRepo{}
	solver := &fakeSolver{}
	svc := zoo.NewZooService(repo, solver)

	_, err := svc.AssignEnclosures(ctx, 3)
	require.NoError(t, err)
	require.Same(t, repo.ctxSeen, ctx)
	require.Same(t, solver.ctxSeen, ctx)
}
