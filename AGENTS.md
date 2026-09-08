# Agent guidance for deadman

Deadman is heartbeat (dead man's switch) monitoring: you register a service
with an expected check-in interval, your service hits its check-in URL on
that schedule, and if it goes quiet past its grace period Deadman emails you.
It emails again when the service checks back in.

## Stack

- Go 1.26, module `github.com/ryanburnette/deadman`
- Entry point: `cmd/deadman/main.go` (CLI dispatch, subcommands in
  `serve.go` and `service.go`)
- HTTP router: chi v5
- Email: stdlib `net/smtp` (STARTTLS on 587/25, implicit TLS on 465)
- Load `skill:go-develop` for conventions (project layout, error handling)
  before adding features — this file only has invariants that skill
  doesn't cover.

## Config is CLI-managed, never hand-edited

`services.csv` (path via `-config` / `CONFIG_PATH`, default
`~/.config/deadman/services.csv`) is the list of monitored services: name,
token, interval, grace, repeat, email. It's managed exclusively through
`deadman add|set|remove|list|url` — always use those, don't write the CSV
by hand or by script, since the CLI is what generates check-in tokens and
keeps the format consistent. The CLI creates the parent directory if it
doesn't exist yet.

A running `deadman serve` polls this file's mtime (every `-check-interval`,
default 15s) and reloads it live, so CLI edits take effect without a
restart. New services get a fresh grace period starting at the moment
they're first loaded (CLI add time, or server start/reload time), not
retroactively.

`state.json` (path via `-state` / `STATE_PATH`, default
`~/.local/deadman/state.json`) is separate: last-seen timestamp, up/down
status, last-notified time per service. It's runtime data the server
owns — don't hand-edit it either. It's what makes a restart not forget an
in-progress outage.

## CLI usage

```sh
deadman add -interval 5m -grace 1m [-repeat 1h] -email you@example.com <name>
deadman set -interval 10m -email new@example.com <name>
deadman remove <name>
deadman list
deadman url <name>          # prints the check-in URL (needs PUBLIC_URL for a full URL)
deadman serve               # starts the HTTP server + check loop
```

Flags come before the positional `<name>` argument — stdlib `flag` stops
parsing at the first non-flag arg, so `deadman add web1 -interval 5m` will
NOT work as expected; `deadman add -interval 5m web1` will.

`-repeat 0` (the default) means alert once on down and stay silent until
recovery. A nonzero `-repeat` resends the down alert at that interval for
as long as the service stays down.

## Pre-commit

```sh
go fmt ./...
go build ./...
go vet ./...
go test ./...
```

## Running

```sh
go run ./cmd/deadman serve
```

Reads `.env` in development (see `.env.example`). Requires `SMTP_HOST`,
`SMTP_PORT`, `SMTP_FROM` (and optionally `SMTP_USER`/`SMTP_PASS`) to start.
`ADDR` defaults to `:3080`.
