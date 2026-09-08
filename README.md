# deadman

A dead man's switch for your services.

You tell deadman about a service and how often it's supposed to check in.
Your service hits a check-in URL on that schedule — a plain `GET` request
is all it takes. If deadman doesn't hear from it within the expected
interval plus a grace period, it emails you. When the service checks in
again, it emails you that too.

No dashboard, no database, no dependencies beyond a mail server. Two small
files hold everything: a config file listing your services, and a state
file tracking the last time each one checked in.

## Install

```sh
go install github.com/ryanburnette/deadman/cmd/deadman@latest
```

Or clone and build:

```sh
git clone https://github.com/ryanburnette/deadman.git
cd deadman
go build -o deadman ./cmd/deadman
```

## Quick start

1. Set your SMTP details as environment variables (or in a `.env` file
   next to wherever you run deadman — copy `.env.example` to start):

   ```sh
   export SMTP_HOST=smtp.example.com
   export SMTP_PORT=587
   export SMTP_USER=you@example.com
   export SMTP_PASS=your-password
   export SMTP_FROM=deadman@example.com
   ```

2. Register a service:

   ```sh
   deadman add -interval 5m -grace 1m -email you@example.com my-service
   ```

   This prints a check-in URL (path only, unless `PUBLIC_URL` is set — see
   below). Something like:

   ```
   check-in path: /checkin/my-service/9dcf9a1e055424af89b8502093fa1cbc
   ```

3. Have `my-service` hit that URL on your server, on schedule — a cron
   job, a line at the end of a backup script, a health-check ping baked
   into a long-running process, whatever fits.

4. Start the server:

   ```sh
   deadman serve
   ```

If `my-service` goes quiet for longer than its interval plus its grace
period, you get an email. When it checks in again, you get another one
saying it recovered.

## Configuring services

Everything about a service — its schedule, its notification email, its
check-in token — is set through the CLI, not by hand-editing files:

```sh
deadman add [options] <name>     # register a new service
deadman set [options] <name>     # change an existing service's settings
deadman remove <name>            # stop monitoring a service
deadman list                     # show all services and their current status
deadman url <name>               # print a service's check-in URL again
```

`add` options:

| Flag | Meaning | Default |
|---|---|---|
| `-interval` | how often the service should check in (e.g. `5m`, `1h`) | required |
| `-grace` | how much extra time to allow before alerting | `1m` |
| `-repeat` | resend the down alert at this interval while still down | `0` (alert once, then stay silent until recovery) |
| `-email` | where to send alerts for this service | required |

`set` takes the same flags to change them, plus `-regen-token` to rotate a
service's check-in token (e.g. if it leaked). Flags must come *before* the
service name: `deadman add -interval 5m my-service`, not
`deadman add my-service -interval 5m`.

Service definitions live in `~/.config/deadman/services.csv` by default;
heartbeat history lives in `~/.local/deadman/state.json`. Override either
with `-config`/`-state` flags or `CONFIG_PATH`/`STATE_PATH` environment
variables. Both files are created automatically the first time you need
them.

You can add, edit, or remove services while `deadman serve` is running —
it picks up changes to the config file automatically, no restart needed.

## Environment variables

| Variable | Purpose | Default |
|---|---|---|
| `SMTP_HOST` | mail server host | required to run `serve` |
| `SMTP_PORT` | mail server port (`465` uses implicit TLS, anything else uses STARTTLS) | required to run `serve` |
| `SMTP_FROM` | from address for alert emails | required to run `serve` |
| `SMTP_USER` | SMTP auth username | none (no auth) |
| `SMTP_PASS` | SMTP auth password | none |
| `PUBLIC_URL` | base URL the CLI prepends when printing check-in URLs, e.g. `https://deadman.example.com` | unset (prints path only) |
| `CONFIG_PATH` | services config file path | `~/.config/deadman/services.csv` |
| `STATE_PATH` | state file path | `~/.local/deadman/state.json` |
| `ADDR` | address `serve` listens on | `:3080` |

## Running the server

```sh
deadman serve [-addr :3080] [-config path] [-state path] [-check-interval 15s]
```

`-check-interval` controls how often deadman checks for missed heartbeats
*and* re-reads the config file for changes.

deadman exposes:

- `GET /checkin/{name}/{token}` — the check-in endpoint your services hit
- `GET /healthz` — for your own uptime checks on deadman itself

See `AGENTS.md` for internals and file formats if you're extending this.
