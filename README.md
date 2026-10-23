# Letterbox

A REST API backend for a social movie-rating platform. Users can register, sign in
with cookie-based sessions, manage their profile, follow other users, block or
unblock them, and create, update, delete, and list reviews. Reviews can be listed per
user, per movie, and as a feed of the users they follow. The data model is prepared
for comments and reactions; those endpoints are not implemented yet.

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
  domains/              Domain models (User, Session, Review, PublicUser)
  handlers/             HTTP handlers
  middleware/           Authentication and rate limiting
  repositories/         SQL queries
  router/               Route definitions
  server/               HTTP server, timeouts, graceful shutdown
  services/             Business logic and validation
  testutil/             Shared fakes for unit tests
```

Requests flow through `router -> handler -> service -> repository`. Each domain has its
own repository (`UserRepository`, `SessionRepository`, `FollowRepository`,
`BlockRepository`, `ReviewRepository`, `MovieRepository`) so SQL stays out of handlers
and services. `MovieRepository` is currently an empty stub; movies are stored in a
local `movies` table keyed by TMDB id, and no TMDB client exists yet.

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
| `TMDB_API_KEY` | placeholder | Reserved for future TMDB integration; not read yet |

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
| `GET` | `/users/username/{username}` | No | Implemented |
| `GET` | `/users/username/{username}/followers` | No | Implemented |
| `GET` | `/users/username/{username}/following` | No | Implemented |
| `POST` | `/users/username/{username}/follow` | Yes | Implemented |
| `DELETE` | `/users/username/{username}/unfollow` | Yes | Implemented |
| `POST` | `/users/username/{username}/block` | Yes | Implemented |
| `DELETE` | `/users/username/{username}/block` | Yes | Implemented |
| `GET` | `/users/username/{username}/followers/count` | No | Implemented |
| `GET` | `/users/username/{username}/following/count` | No | Implemented |
| `GET` | `/users/username/{username}/reviews` | Yes | Implemented |
| `GET` | `/reviews/following` | Yes | Implemented |
| `GET` | `/movies/{movie_id}/reviews` | No | Implemented |
| `POST` | `/movies/{movie_id}/reviews` | Yes | Implemented |
| `PATCH` | `/movies/{movie_id}/reviews` | Yes | Implemented |
| `DELETE` | `/movies/{movie_id}/reviews` | Yes | Implemented |

Follower/following lists, a user's reviews, the following feed, and a movie's reviews
accept `limit` (1-100, default 20) and `offset` (default 0). Count endpoints return
`{"count": N}`. Follow and block actions return `204 No Content` on success.

Reviews are returned newest-updated first as a JSON array of review objects (`id`,
`user_id`, `movie_id`, `score`, `content`, `containsSpoilers`, `created_at`,
`updated_at`), with `id` as the tiebreaker. `GET /reviews/following` returns reviews
from every user the authenticated user follows. If a block exists in either direction,
`GET /users/username/{username}/reviews` returns `403 Forbidden`.
`GET /movies/{movie_id}/reviews` is public and returns `400 Bad Request` for movie IDs
that are not positive 32-bit integers.

`POST /movies/{movie_id}/reviews` creates a review for the authenticated user. `score`
is required and must be between 1 and 10; `content` may be empty and is capped at
10,000 characters; `containsSpoilers` defaults to `false`. It returns `201 Created`
with the review, `400` for invalid input, and `409` if the user already reviewed that
movie. The movie must already exist in the local `movies` cache; missing movies
currently surface as `500` because foreign-key violations are not mapped.

`PATCH /movies/{movie_id}/reviews` partially updates the authenticated user's review.
Any subset of `score`, `content`, and `containsSpoilers` may be sent, but at least one
is required; omitted fields keep their current values. A successful update returns
`200 OK` with the review and a refreshed `updated_at`. It returns `400` for invalid
input or an empty field set, and `404` if the user has not reviewed that movie.

`DELETE /movies/{movie_id}/reviews` deletes the authenticated user's review and
returns `204 No Content`. It returns `404` if the user has not reviewed that movie.
The movie ID is validated on both routes and returns `400` when it is not a positive
32-bit integer.

### Request body limits

JSON request bodies are capped and oversized bodies return `413 Request Entity Too
Large`; malformed or non-JSON bodies return `400 Bad Request`.

| Endpoint | Limit |
| --- | --- |
| `POST /auth/register` | 1 KiB |
| `POST /auth/login` | 1 KiB |
| `PUT /users/me` | 4 KiB |
| `PUT /users/me/password` | 1 KiB |
| `POST /movies/{movie_id}/reviews` | 64 KiB |
| `PATCH /movies/{movie_id}/reviews` | 64 KiB |

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

Follow and unfollow another user:

```bash
curl -i -b cookies.txt -X POST http://localhost:8080/api/v1/users/username/grace/follow
curl -i -b cookies.txt -X DELETE http://localhost:8080/api/v1/users/username/grace/unfollow
```

Block and unblock another user:

```bash
curl -i -b cookies.txt -X POST http://localhost:8080/api/v1/users/username/grace/block
curl -i -b cookies.txt -X DELETE http://localhost:8080/api/v1/users/username/grace/block
```

List a user's reviews (newest updated first):

```bash
curl -i -b cookies.txt "http://localhost:8080/api/v1/users/username/ada/reviews?limit=10&offset=0"
```

Read reviews from users you follow:

```bash
curl -i -b cookies.txt "http://localhost:8080/api/v1/reviews/following?limit=10&offset=0"
```

Create a review (the movie must already be in the local cache):

```bash
curl -i -b cookies.txt -X POST http://localhost:8080/api/v1/movies/550/reviews \
  -H 'Content-Type: application/json' \
  -d '{"score":8,"content":"Great movie","containsSpoilers":false}'
```

Partially update that review (only the fields sent are changed):

```bash
curl -i -b cookies.txt -X PATCH http://localhost:8080/api/v1/movies/550/reviews \
  -H 'Content-Type: application/json' \
  -d '{"score":9,"containsSpoilers":true}'
```

Delete that review:

```bash
curl -i -b cookies.txt -X DELETE http://localhost:8080/api/v1/movies/550/reviews
```

## Follows and blocks

Following yourself is rejected with `400`; following the same user twice returns `409`.
Unfollowing someone you do not follow also returns `409`.

Blocking is transactional: it removes any follow relationship in either direction and
creates the block in a single transaction. While a block exists in either direction,
following is rejected with `403 Forbidden`. Blocking yourself returns `400`, blocking an
already-blocked user returns `409`, and unblocking a user you do not block returns `409`.
Unblocking does **not** restore previous follow relationships.

Blocks also hide reviews in both directions: neither user can list the other's reviews
through `GET /users/username/{username}/reviews` (both get `403 Forbidden`). The
following feed never includes a blocked user because blocking removes the underlying
follow relationship.

## Authentication

Registration validates the username (3-20 letters, numbers, or underscores), email,
password (8-72 bytes), and country (optional two-letter code), then stores the password
as a bcrypt hash (cost 12). Hashes are never returned in API responses.

Login accepts an email or username and sets an `HttpOnly`, `SameSite=Strict` `session_id`
cookie. Sessions are stored in PostgreSQL, last **1 hour**, and are deleted hourly by a
background cleanup job. Logout deletes only the current session and clears the cookie;
deleting an account removes all of the user's sessions and data via cascading foreign keys.
Changing the password requires the current password and invalidates every other session.

## Rate limiting

Every request is limited to 100 requests per minute per client IP. Exceeding the limit
returns `429 Too Many Requests` with a `Retry-After` header.

## Testing

```bash
go test ./...
go test -race ./...
```

Unit tests cover services (including review listing, creation, update, and deletion),
handlers (including JSON body limits), middleware, and repository helpers, and use fakes
from `internal/testutil`. Repository integration tests are not included because they
require a running PostgreSQL instance.

## Notes and limitations

- Review listing (per user, following feed, and per movie), creation, partial updates,
  and deletion are implemented. `MovieRepository` has no methods, so creating a review
  requires the movie to already exist in the local `movies` cache.
- `TMDB_API_KEY` is present in `.env.example`, but nothing reads it yet.
- Tables for movies, comments, and reactions exist; comments and reactions have no code.
- Blocks prevent follows, remove existing follow relationships, and hide reviews in both
  directions; they are not yet used to hide profiles, follower/following lists, comments,
  or reactions.
- There is no lint configuration in the repository yet.
