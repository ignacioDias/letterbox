# AGENTS.md

## Build and verify

- Go 1.27.1. There is a unit test suite (`go test ./...`), but no linter config, Makefile, or CI. Do not invent commands.
- Run after every change:
  ```bash
  gofmt -l .
  go build ./...
  go vet ./...
  ```
  `gofmt -l .` must print nothing.
- Local run needs `.env` (gitignored; copy `.env.example`):
  ```bash
  docker compose up -d   # Postgres exposed on host port 5433, not 5432
  go run ./cmd/letterbox
  ```
- The compiled `letterbox` binary in the repo root is a build artifact; never commit it.

## Architecture

- Strict layering: `router -> handler -> service -> repository`. Handlers do not run SQL, repositories do not validate, services own validation and business rules.
- `internal/database/database.go` wires repositories into the `Database` struct. Constructors return interface types; keep each interface next to its implementation.
- Entry point `cmd/letterbox/main.go`: loads env via godotenv, connects with sqlx/pgx, runs `database.Init()`, starts hourly expired-session cleanup, then the server with graceful shutdown.
- Schema is in `internal/database/schema.go` and is applied on every startup in one transaction using `CREATE TABLE IF NOT EXISTS`. There are no migrations: add statements to `SchemaStatements` to change tables.
- Routes in `internal/router/router.go` use Go method patterns. Read path params with `r.PathValue("username")`; do not reintroduce the old `/users/username/{username}` paths.

## Auth and errors

- Protect routes by wrapping the handler in `r.auth.Authenticate`. It reads the `session_id` cookie, validates the session in Postgres, and stores the user id in context; read it with `middleware.GetUserID`.
- Sessions last 1 hour and are rejected once expired. Logout deletes only the current session and clears the cookie; account deletion cascades all user data and also clears the cookie via `clearSessionCookie`. Changing a password deletes every other session but keeps the current one.
- Services and repositories expose sentinel errors (e.g. `ErrInvalidUsername`, `ErrUserNotFound`, `ErrUsernameAlreadyExists`); handlers map them to status codes. Keep this split for new endpoints (400 invalid, 404 missing, 409 conflict).
- `PUT /users/me` treats an empty string as "keep existing value", so fields cannot currently be cleared.
- A global limiter allows 100 requests/minute per client IP; expect `429` during repeated manual testing.

## Current gaps

- Follow/unfollow, blocks, and follower/following lists and counts are implemented.
- Tables for reviews, comments, and reactions exist in the schema, but no code uses them yet.
