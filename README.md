# Zoo Enclosure Assignment API

A small Go REST service that assigns a fixed roster of fictional animals to a
requested number of enclosures (1..8), honouring per-animal incompatibility
and `maximumFriends` capacity constraints.

One endpoint: `GET /api/v1/zoos/{enclosures}`. Plus `/healthz` and `/readyz`.

---

## Prerequisites

- Go 1.25+ (only needed for local builds; the Docker path is self-contained)
- Docker (for `docker compose` and the integration tests)
- `make`

---

## Architecture

Dependency direction is strict and one-way:

```
   interfaces (chi, dto, httperr)
          │
          ▼
       application (ZooService, Repository + Solver ports)
          ▲
          │
   infrastructure (postgres / memory repo, greedy solver, seeder)
          │
          ▼
        domain (Animal, Enclosure — zero intra-repo imports)
```

`cmd/api/main.go` is the only place that knows concrete adapters exist. Every
other package depends on interfaces owned by its caller.

---

## Run via Docker

```sh
cp .env.example .env          # edit values if you want
docker compose up --build     # api + postgres
```

Stack smoke:

```sh
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
curl -s localhost:8080/api/v1/zoos/3 | jq
```

Tear down (keep data): `docker compose down`
Tear down (wipe data): `docker compose down -v`

---

## Run locally (in-memory fallback)

If `POSTGRES_URL` is empty, the service loads the embedded `animals.json` into
an in-memory repository (no Postgres needed).

```sh
POSTGRES_URL= make run
# …in another shell…
curl -s localhost:8080/api/v1/zoos/3 | jq
```

Or build the binary:

```sh
make build
POSTGRES_URL= ./bin/api
```

---

## Migrations and the seeder

Migrations are SQL files in [`migrations/`](migrations/) embedded into the
binary via `//go:embed`. On boot with `POSTGRES_URL` set, the service runs:

```
migrate up  →  seed (idempotent)  →  serve
```

Manual:

```sh
make migrate-up       # go run ./cmd/api migrate up
make migrate-down     # go run ./cmd/api migrate down
```

The seeder loads `internal/infrastructure/seed/animals.json` only if the
`animals` table is empty. Running it twice is a no-op (row count stays at 100).

---

## Tests

```sh
make test        # unit only, -race
make test-int    # integration, //go:build integration, spins postgres:16-alpine
                  #   via testcontainers-go (Docker must be running)
make cover       # HTML coverage report; fails if total < 80 %
make lint        # gofmt -l check + golangci-lint v2
make fmt         # gofmt + goimports rewrite
```

---

## Switching solvers

The `Solver` port has one method (`Assign`). The implementation is selected at
startup via the `SOLVER` env var and resolved by `NewSolver` in
[`internal/infrastructure/solver/factory.go`](internal/infrastructure/solver/factory.go).
`greedy` is the only shipped solver; adding another is **one new package
(implementing the port) + one factory case**, with no edits anywhere else.

```sh
SOLVER=greedy     make run   # default
SOLVER=maxpop     make run   # returns "unknown solver" at startup; not built yet
```

Every future solver must pass the `SolverContract` suite in
[`internal/application/zoo/zoocontract`](internal/application/zoo/zoocontract).

---

## Example requests

Happy path:

```sh
curl -s localhost:8080/api/v1/zoos/3 | jq
```

Validation errors (all return 400 `application/problem+json`):

```sh
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/zoos/0
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/zoos/9
curl -s -o /dev/null -w "%{http_code}\n" localhost:8080/api/v1/zoos/abc
```

Liveness and readiness:

```sh
curl -s localhost:8080/healthz
curl -s localhost:8080/readyz
```

---

## Project layout

```
cmd/api/                         # Composition root (main.go)
internal/
  domain/                        # Enterprise rules; zero intra-repo imports
    animal/                      #   Animal value object, invariants
    zoo/                         #   Enclosure aggregate
  application/zoo/               # Use case + consumer-owned ports
    zoocontract/                 #   Reusable Solver contract test suite
  infrastructure/
    persistence/memory/          # In-memory Repository adapter
    persistence/postgres/        # pgxpool Repository adapter (JSONB)
    solver/                      # NewSolver factory
      greedy/                    #   Default greedy Solver
    seed/                        # Embedded roster + idempotent seeder
  interfaces/http/
    router/                      # chi routes + versioned sub-routers
    handler/                     # Thin handlers: parse → service → render
    middleware/                  # Recoverer, request logger
    dto/                         # Wire shapes + domain converters
    httperr/                     # Problem+json catalogue + Map()
  platform/
    config/                      # envconfig Config struct
    logging/                     # slog setup + context accessors
    migrate/                     # goose wrapper; embeds migrations/*.sql
migrations/                      # Goose SQL migrations (embedded into binary)
test/integration/                # testcontainers-backed end-to-end tests
```

---

## Assumptions & Trade-offs

Explicitly noted so reviewers don't have to guess what was scoped out:

- **Greedy solver only.** The `Solver` port is set up so a smarter
  implementation (branch-and-bound, popularity-maximising, etc.) is one new
  package with no edits elsewhere. The greedy solver skips animals that
  don't fit (capacity or incompatibility); each animal is placed at most
  once.
- **`/api/v1/zoos/{enclosures}` can return a partial assignment.** The
  embedded 100-animal roster has tight `maximumFriends` values, so no
  algorithm fits every animal into 1..8 enclosures. The endpoint returns 200
  with a non-empty assignment of as many animals as fit; the rest are
  dropped. A 422 `infeasible-assignment` response is reserved for future
  solvers that refuse on infeasibility.
- **Vendored dependencies.** `vendor/` is committed. The local Docker Desktop
  MITMs `proxy.golang.org` with a cert the container doesn't trust; vendoring
  lets the image build offline. `go mod tidy` is still the source of truth.
- **Single endpoint, single resource.** No `/api/v2` scaffolding. The design
  supports versioned sub-routers; nothing is built yet.
- **Scope "Won't haves":** no auth, no rate limiting, no OpenAPI/Swagger, no
  Prometheus/OTEL, no Kubernetes manifests, no pprof/load-test scripts, no
  ORM.

---

## Secrets & configuration

All configuration is environment-driven (`envconfig`, 12-factor). See
[internal/platform/config/config.go](internal/platform/config/config.go) for
the canonical struct. No flag parsing, no config files, no secrets in source.

- **Local dev.** Copy `.env.example` → `.env` and edit. `.env` is gitignored;
  `.env.example` is committed with placeholder values only.
- **Production.** Secrets come from the platform secret manager (AWS Secrets
  Manager, Vault, whatever the deploy target provides). The service only
  reads `os.Getenv`; it does not care where the value came from. The
  `platform/config` package is the **only** place in the codebase that calls
  into env reading.
- **Logging.** Connection strings and tokens are never logged, even at debug.
  The postgres pool constructor returns scrubbed errors so driver-level
  messages don't leak credentials.

### Env vars

```
APP_ENV                           dev | staging | prod         (default: dev)
HTTP_ADDR                         :8080
HTTP_READ_TIMEOUT                 5s
HTTP_WRITE_TIMEOUT                10s
HTTP_SHUTDOWN_GRACE               15s
POSTGRES_URL                      empty → in-memory fallback
POSTGRES_MAX_CONNS                10
POSTGRES_MIN_CONNS                2
POSTGRES_MAX_CONN_LIFETIME        1h
POSTGRES_MAX_CONN_IDLE_TIME       30m
SOLVER                            greedy
LOG_LEVEL                         debug | info | warn | error  (default: info)
```

Additional vars used by `docker-compose.yml`: `POSTGRES_USER`,
`POSTGRES_PASSWORD`, `POSTGRES_DB` (for the Postgres container bootstrap).
