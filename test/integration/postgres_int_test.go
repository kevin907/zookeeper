//go:build integration

// Package integration holds testcontainers-backed end-to-end tests. The build
// tag keeps them out of the default `go test ./...` run.
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	appzoo "github.com/kevintrivedi/zoo-api/internal/application/zoo"
	"github.com/kevintrivedi/zoo-api/internal/domain/animal"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/persistence/memory"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/persistence/postgres"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/seed"
	"github.com/kevintrivedi/zoo-api/internal/infrastructure/solver/greedy"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/dto"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/handler"
	"github.com/kevintrivedi/zoo-api/internal/interfaces/http/router"
	"github.com/kevintrivedi/zoo-api/internal/platform/migrate"
)

// ensure nat import stays referenced even on older testcontainers APIs.
var _ = nat.Port("5432/tcp")

func TestPostgres_MigrateSeedList_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool := startPostgres(t, ctx)

	// Up → Down → Up: migrations are reversible.
	require.NoError(t, migrate.Up(ctx, pool), "first up")
	require.NoError(t, migrate.Down(ctx, pool), "down")
	require.NoError(t, migrate.Up(ctx, pool), "second up")

	repo := postgres.New(pool)

	// Seed twice: idempotent (count stays at 100).
	require.NoError(t, seed.Seed(ctx, repo))
	require.NoError(t, seed.Seed(ctx, repo))

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 100, count)

	// Repository round-trip matches the embedded roster.
	animals, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, animals, 100)
	require.Equal(t, "snake", animals[0].Type)
	require.Equal(t, 10, animals[0].Popularity)

	// Ping is cheap and reflects pool health.
	require.NoError(t, repo.Ping(ctx))

	// HTTP end-to-end with a feasible roster. The full roster can be
	// infeasible for greedy at low N because some animals have
	// maximumFriends=1.
	feasible := memory.New(feasibleRoster(t))
	svc := appzoo.NewZooService(feasible, greedy.New())
	h := router.New(
		handler.NewZooHandler(svc),
		repo,
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		2*time.Second,
		250*time.Millisecond,
	)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/zoos/3", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))

	var body dto.ZooResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Enclosures, 3)
	for i, enc := range body.Enclosures {
		require.NotEmpty(t, enc.Animals, "enclosure %d must not be empty", i)
	}
	require.NotNil(t, body.Unplaced, "response must always include an unplaced array")

	// Validation path: N out of range returns 400 problem+json.
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/api/v1/zoos/9", nil))
	require.Equal(t, http.StatusBadRequest, rec2.Code)
	require.Equal(t, "application/problem+json; charset=utf-8", rec2.Header().Get("Content-Type"))

	// Readiness: real Postgres is up, /readyz responds 200.
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	require.Equal(t, http.StatusOK, rec3.Code)
}

func feasibleRoster(t *testing.T) []animal.Animal {
	t.Helper()
	specs := []struct {
		typ        string
		popularity int
		maxFriends int
	}{
		{"lion", 10, 20}, {"monkey", 9, 20}, {"giraffe", 8, 20},
		{"zebra", 7, 20}, {"lion", 6, 20}, {"monkey", 5, 20},
	}
	out := make([]animal.Animal, 0, len(specs))
	for i, s := range specs {
		id := fmt.Sprintf("feasible-%02d", i)
		a, err := animal.New(id, s.typ, s.popularity, s.maxFriends, nil)
		require.NoError(t, err)
		out = append(out, a)
	}
	return out
}

func startPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("zoo"),
		tcpostgres.WithUsername("zoo"),
		tcpostgres.WithPassword("zoo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90*time.Second),
		),
	)
	require.NoError(t, err, "start postgres container")
	t.Cleanup(func() {
		_ = testcontainers.TerminateContainer(container)
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	return pool
}
