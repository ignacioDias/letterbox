# Letterbox

A Letterbox copy: register, follow users, rate movies (1–10), and get a feed with
everything the people you follow are doing. You can like, dislike, and comment on
posts, message other users, and block users.

Messaging rules: if you don't follow a user you can send only one message and then
have to wait for them to allow the chat.

- **Backend:** Go (`net/http`)
- **Database:** PostgreSQL (Docker Compose)
- **Deployment:** Docker (planned)

## Status

Early scaffold. The HTTP server, routing entry point, graceful shutdown, rate
limiting, user registration and login, and local PostgreSQL container are in place;
most domain routes and full deployment are not implemented yet.

## Project structure

```
cmd/letterbox/       Application entry point
internal/router/     Route registration (http.ServeMux)
internal/server/     HTTP server setup and graceful shutdown
internal/middleware/ HTTP middleware (rate limiting)
internal/database/   Database connection and schema
compose.yaml         Local PostgreSQL container
```

## Requirements

- Go 1.27+
- Docker with the Compose plugin

## Getting started

Start PostgreSQL:

```sh
cp .env.example .env   # first time only
docker compose up -d db
```

Then run the application:

```sh
go run ./cmd/letterbox
```

The server listens on port `8080` and shuts down gracefully on `SIGINT`/`SIGTERM`
with a 30-second timeout. The schema is applied automatically on startup.

Useful commands:

```sh
docker compose logs -f db    # follow database logs
docker compose down          # stop the database
docker compose down -v       # stop and delete all data
```

## Configuration

Environment variables are loaded from `.env` (ignored by git; do not commit
secrets). Docker Compose reads the same file for the database container.

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | HTTP listen port | `8080` |
| `APP_ENV` | Set to `production` to enable secure cookies | `development` |
| `DATABASE_URL` | PostgreSQL connection string | — |
| `POSTGRES_USER` | Database user | `letterbox` |
| `POSTGRES_PASSWORD` | Database password | `letterbox` |
| `POSTGRES_DB` | Database name | `letterbox` |
| `POSTGRES_PORT` | Host port mapped to PostgreSQL | `5433` |

## Authentication

| Method | Endpoint | Status |
| --- | --- | --- |
| `POST` | `/api/v1/auth/register` | Implemented |
| `POST` | `/api/v1/auth/login` | Implemented |
| `DELETE` | `/api/v1/auth/logout` | Not implemented |
| `GET` | `/api/v1/auth/me` | Not implemented |

Registration validates the username, email, and password, then stores the password as
a bcrypt hash (cost 12). The hash is never returned in API responses.

Login accepts an email or username plus password and sets an `HttpOnly`,
`SameSite=Strict` `session_id` cookie. Sessions last **1 hour** and are stored in the
`sessions` table. Sets `Secure` when `APP_ENV=production`.

Expired sessions are deleted when the server starts and then every hour.

## Rate limiting

All routes are limited to **100 requests per minute per IP**, enforced in-memory by
`internal/middleware/rate_limit.go`. Exceeding the limit returns `429 Too Many
Requests` with a `Retry-After` header.

The limit is per server process and resets when the server restarts. Client IPs come
from `RemoteAddr`; when running behind a reverse proxy, forwarded-header support must
be added so all clients aren't limited under the proxy's IP.

## Planned features

- [x] User registration and login with sessions
- [ ] Logout, current-user endpoint, and session validation middleware
- [ ] Follow / unfollow and blocking
- [ ] Movie ratings (1–10)
- [ ] Activity feed of followed users
- [ ] Likes, dislikes, and comments on posts
- [ ] Direct messaging with follow-gated chat requests
- [ ] PostgreSQL schema and migrations
- [x] Rate limiting middleware
- [x] Docker Compose setup for local PostgreSQL
- [ ] Docker deployment of the application

## Development

```sh
go build ./...    # compile
go test ./...     # run tests
go vet ./...      # static checks
```
