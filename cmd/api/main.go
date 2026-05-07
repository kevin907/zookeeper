// Command api is the Zoo Enclosure Assignment service entry point.
// main stays thin: it wires config, logger, repo, solver, router, and
// HTTP server, then blocks on shutdown signals.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	appzoo "github.com/kevin907/zookeeper/internal/application/zoo"
	"github.com/kevin907/zookeeper/internal/infrastructure/persistence/memory"
	pgrepo "github.com/kevin907/zookeeper/internal/infrastructure/persistence/postgres"
	"github.com/kevin907/zookeeper/internal/infrastructure/seed"
	"github.com/kevin907/zookeeper/internal/infrastructure/solver"
	"github.com/kevin907/zookeeper/internal/interfaces/http/handler"
	"github.com/kevin907/zookeeper/internal/interfaces/http/router"
	"github.com/kevin907/zookeeper/internal/platform/config"
	"github.com/kevin907/zookeeper/internal/platform/logging"
	"github.com/kevin907/zookeeper/internal/platform/migrate"
)

const readyzTimeout = 500 * time.Millisecond

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}

	logger := logging.New(os.Stdout, cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if len(args) > 0 && args[0] == "migrate" {
		return runMigrate(ctx, logger, cfg, args[1:])
	}
	if len(args) > 0 && args[0] == "healthcheck" {
		return runHealthcheck(ctx, cfg)
	}
	return serve(ctx, logger, cfg)
}

// runHealthcheck is the probe invoked by docker-compose via
// ["/api", "healthcheck"]. It reaches /readyz over loopback under a 2s
// timeout and exits 0 on HTTP 200, non-zero otherwise. Distroless has
// no shell/curl, so the binary has to probe itself.
func runHealthcheck(ctx context.Context, cfg config.Config) int {
	addr := cfg.HTTPAddr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	url := "http://" + addr + "/readyz"

	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: build request: %v\n", err)
		return 1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "healthcheck: readyz returned %d\n", resp.StatusCode)
		return 1
	}
	return 0
}

func serve(ctx context.Context, logger *slog.Logger, cfg config.Config) int {
	repo, pinger, cleanup, err := buildRepo(ctx, logger, cfg)
	if err != nil {
		logger.Error("build repository", "error", err)
		return 1
	}
	defer cleanup()

	sv, err := solver.NewSolver(cfg.Solver)
	if err != nil {
		logger.Error("build solver", "error", err)
		return 1
	}

	svc := appzoo.NewZooService(repo, sv)
	h := router.New(
		handler.NewZooHandler(svc),
		pinger,
		logger,
		cfg.HTTPReadTimeout,
		readyzTimeout,
	)
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      h,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  60 * time.Second,
	}
	return runServer(ctx, logger, srv, cfg.HTTPShutdownGrace)
}

func runServer(ctx context.Context, logger *slog.Logger, srv *http.Server, grace time.Duration) int {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			logger.Error("http server crashed", "error", err)
			return 1
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), grace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown", "error", err)
		return 1
	}
	logger.Info("stopped cleanly")
	return 0
}

func buildRepo(ctx context.Context, logger *slog.Logger, cfg config.Config) (appzoo.Repository, handler.Pinger, func(), error) {
	if cfg.PostgresURL == "" {
		logger.Warn("POSTGRES_URL empty: falling back to in-memory repository")
		animals, err := seed.Roster()
		if err != nil {
			return nil, nil, func() {}, fmt.Errorf("parse embedded roster: %w", err)
		}
		repo := memory.New(animals)
		return repo, noopPinger{}, func() {}, nil
	}

	pool, err := pgrepo.NewPool(ctx, cfg)
	if err != nil {
		return nil, nil, func() {}, fmt.Errorf("build pgxpool: %w", err)
	}
	if err := migrate.Up(ctx, pool); err != nil {
		pool.Close()
		return nil, nil, func() {}, fmt.Errorf("run migrations: %w", err)
	}
	repo := pgrepo.New(pool)
	if err := seed.Seed(ctx, repo); err != nil {
		pool.Close()
		return nil, nil, func() {}, fmt.Errorf("seed animals: %w", err)
	}
	cleanup := func() { pool.Close() }
	return repo, repo, cleanup, nil
}

// runMigrate handles the ./api migrate up|down subcommand.
func runMigrate(ctx context.Context, logger *slog.Logger, cfg config.Config, rest []string) int {
	if cfg.PostgresURL == "" {
		logger.Error("migrate requires POSTGRES_URL")
		return 2
	}
	if len(rest) != 1 || (rest[0] != "up" && rest[0] != "down") {
		logger.Error("usage: api migrate up|down")
		return 2
	}
	pool, err := pgrepo.NewPool(ctx, cfg)
	if err != nil {
		logger.Error("open pool", "error", err)
		return 1
	}
	defer pool.Close()

	var migErr error
	switch rest[0] {
	case "up":
		migErr = migrate.Up(ctx, pool)
	case "down":
		migErr = migrate.Down(ctx, pool)
	}
	if migErr != nil {
		logger.Error("migrate", "direction", rest[0], "error", migErr)
		return 1
	}
	logger.Info("migrate ok", "direction", rest[0])
	return 0
}

// noopPinger answers /readyz green in the memory-fallback path.
type noopPinger struct{}

func (noopPinger) Ping(context.Context) error { return nil }
