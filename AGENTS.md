# Agent guidance for deadman

Deadman is a Go server. Purpose and scope are still being defined — see
conversation/README for current direction, since this file lags decisions
that haven't been distilled into invariants yet.

## Stack

- Go 1.26, module `github.com/ryanburnette/deadman`
- Entry point: `cmd/deadman/main.go`
- HTTP router: chi v5
- Load `skill:go-develop` for conventions (project layout, sqlc, auth,
  error handling) before adding features — this file only has invariants
  that skill doesn't cover.

## Pre-commit

```sh
go fmt ./...
go build ./...
go vet ./...
go test ./...
```

## Running

```sh
go run ./cmd/deadman
```

Reads `.env` in development (see `.env.example`). `ADDR` defaults to
`:3080`.
