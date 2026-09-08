# deadman

A dead man's switch for your services. Register a service with an expected
check-in interval; your service hits its check-in URL on that schedule. If
it goes quiet past its grace period, Deadman emails you. It emails again
when the service checks back in.

## Usage

```sh
cp .env.example .env   # fill in SMTP settings

deadman add -interval 5m -grace 1m -email you@example.com my-service
deadman list
deadman serve
```

Point `my-service` at the printed check-in URL (a plain `GET` on that
schedule is all it needs to do). See `AGENTS.md` for the full CLI
reference and file formats.
