MODULE  := github.com/kevin907/zookeeper
BIN     := ./bin/api

.PHONY: fmt lint test test-int cover build run up down seed migrate-up migrate-down tidy

GOFMT_PATHS := cmd internal test migrations

fmt:
	gofmt -w $(GOFMT_PATHS)
	goimports -w -local $(MODULE) $(GOFMT_PATHS)

lint:
	@if [ -n "$$(gofmt -l $(GOFMT_PATHS))" ]; then \
		echo "gofmt found unformatted files:"; \
		gofmt -l $(GOFMT_PATHS); \
		exit 1; \
	fi
	golangci-lint run ./...

test:
	go test ./... -race

test-int:
	go test -tags=integration ./test/integration/... -race -timeout=120s

cover:
	# Coverage gate is enforced on internal/domain and internal/application
	# (zoocontract is a test helper, excluded).
	go test -race \
		-coverpkg=./internal/domain/...,./internal/application/zoo \
		-coverprofile=coverage.out \
		./internal/domain/... ./internal/application/zoo
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | awk 'END { pct = substr($$NF, 1, length($$NF)-1) + 0; if (pct < 80) { printf "coverage below 80%%: %.1f%%\n", pct; exit 1 } }'

build:
	mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o $(BIN) ./cmd/api

run:
	go run ./cmd/api

up:
	docker compose up --build

down:
	docker compose down

seed:
	go run ./cmd/api seed

migrate-up:
	go run ./cmd/api migrate up

migrate-down:
	go run ./cmd/api migrate down

tidy:
	go mod tidy
	go mod verify
