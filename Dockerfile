# syntax=docker/dockerfile:1.7

FROM golang:1.25-bookworm AS builder
WORKDIR /src

# Copy the whole tree (including vendor/) and build without network access.
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} \
    go build -mod=vendor -trimpath -ldflags="-s -w -buildid=" -o /out/api ./cmd/api

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
