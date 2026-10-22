# Letterbox

A REST API backend for a social movie-rating platform. Users can register, sign in
with cookie-based sessions, manage their profile, and browse follower/following lists.
The data model is already prepared for reviews, comments, reactions, follows, and blocks.

## Tech stack

- **Go 1.27** with the standard `net/http` router (`ServeMux` patterns)
- **PostgreSQL 17** (via Docker Compose)
- **sqlx** + **pgx** for database access
- **bcrypt** for password hashing
- **godotenv** for local environment loading

## Project structure

```
cmd/letterbox/          Application entry point
internal/
  database/             Connection wiring, schema creation
  domains/              Domain models (User, Session, PublicUser)
  handlers/             HTTP handlers
  middleware/           Authentication and rate limiting
  repositories/         SQL queries
  router/               Route definitions
  server/               HTTP server, timeouts, graceful shutdown
  services/             Business logic and validation
```

Requests flow through `router -> handler -> service -> repository`. Each domain has its
own repository (`UserRepository`, `SessionRepository`, `FollowRepository`) so SQL stays
out of handlers and services.

## Getting started

Requirements: Go 1.27+ and Docker.

```bash
cp .env.example .env
docker compose up -d
go run ./cmd/letterbox
```

The API listens on `http://localhost:8080` by default. The schema is created
automatically on startup; there is no separate migration step.

### Environment variables

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | none (required) | PostgreSQL connection string |
| `PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `development` | Set to `production` to mark the session cookie `Secure` |
| `POSTGRES_USER` | `letterbox` | Postgres user for the Compose container |
| `POSTGRES_PASSWORD` | `letterbox` | Postgres password for the Compose container |
| `POSTGRES_DB` | `letterbox` | Postgres database name |
| `POSTGRES_PORT` | `5433` | Host port mapped to the container |

## API

Base path: `/api/v1`

| Method | Path | Auth | Status |
| --- | --- | --- | --- |
| `POST` | `/auth/register` | No | Implemented |
| `POST` | `/auth/login` | No | Implemented |
| `DELETE` | `/auth/logout` | Yes | Implemented |
| `GET` | `/users/me` | Yes | Implemented |
| `PUT` | `/users/me` | Yes | Implemented |
| `PUT` | `/users/me/password` | Yes | Implemented |
| `DELETE` | `/users/me` | Yes | Implemented |
| `GET` | `/users/{username}` | No | Implemented |
| `GET` | `/users/{username}/followers` | No | Implemented |
| `GET` | `/users/{username}/following` | No | Implemented |
| `POST` | `/users/{username}/follow` | Yes | Planned |
| `DELETE` | `/users/{username}/unfollow` | Yes | Planned |
| `GET` | `/users/{username}/followers/count` | No | Planned |
| `GET` | `/users/{username}/following/count` | No | Planned |

Follower/following lists accept `limit` (1-100, default 20) and `offset` (default 0).

### Examples

Register:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"ada","email":"ada@example.com","password":"secret123","country":"AR"}'
```

Login and keep the session cookie:

```bash
curl -i -c cookies.txt -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"emailOrUsername":"ada","password":"secret123"}'
```

Update the current user (omitted fields keep their existing values):

```bash
curl -i -b cookies.txt -X PUT http://localhost:8080/api/v1/users/me \
  -H 'Content-Type: application/json' \
  -d '{"country":"UY"}'
```

Log out:

```bash
curl -i -b cookies.txt -X DELETE http://localhost:8080/api/v1/auth/logout
```

## Authentication

Registration validates the username (3-20 letters, numbers, or underscores), email, and
password (8-72 bytes), then stores the password as a bcrypt hash (cost 12). Hashes are
never returned in API responses.

Login accepts an email or username and sets an `HttpOnly`, `SameSite=Strict` `session_id`
cookie. Sessions are stored in PostgreSQL, last **1 hour**, and are deleted hourly by a
background cleanup job. Logout deletes only the current session and clears the cookie;
deleting an account removes all of the user's sessions and data via cascading foreign keys.
Changing the password requires the current password and invalidates every other session.

## Rate limiting

Every request is limited to 100 requests per minute per client IP. Exceeding the limit
returns `429 Too Many Requests` with a `Retry-After` header.

## Notes and limitations

- Follow/unfollow and follower/following counts are not implemented yet.
- There is no test suite or lint configuration in the repository yet.
