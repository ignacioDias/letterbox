# Letterbox

A REST API backend for a social movie-rating platform. Users can register, sign in
with cookie-based sessions, manage their profile, search other users by username,
follow other users, block or unblock them, and create, update, delete, and list reviews. Reviews can be listed per
user, per movie, and as a feed of the users they follow. Users can also like or dislike reviews.
Movies can be looked up by TMDB id, searched, listed from TMDB's daily trending feed, or
expanded with similar titles, with results cached locally. The data model is prepared for
comments; comment endpoints are not implemented yet.

There is a [vibe coded frontend for this project](https://github.com/ignacioDias/frontend-letterbox).

## Tech stack

- **Go 1.27** with the standard `net/http` router (`ServeMux` patterns)
- **PostgreSQL 17** (via Docker Compose)
- **sqlx** + **pgx** for database access
- **golang-tmdb** for movie metadata, cached in PostgreSQL
- **bcrypt** for password hashing
- **godotenv** for local environment loading

## Project structure

```
cmd/letterbox/          Application entry point
internal/
  database/             Connection wiring, schema creation
  domains/              Domain models (User, Session, Review, Reaction, Movie, PublicUser)
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
`BlockRepository`, `ReviewRepository`, `ReactionRepository`, `MovieRepository`) so SQL
stays out of handlers and services. `MovieRepository` reads and writes a local `movies`
table keyed by TMDB id; on a cache miss, `MovieService` fetches the movie from TMDB and
stores it before returning it.

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
| `TMDB_API_KEY` | none | TMDB API key used when a movie is missing from the local cache |

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
| `GET` | `/users/search` | No | Implemented |
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
| `GET` | `/movies/{movie_id}/reviews/average` | No | Implemented |
| `GET` | `/movies/{movie_id}/reviews/count` | No | Implemented |
| `POST` | `/movies/{movie_id}/reviews` | Yes | Implemented |
| `GET` | `/reviews/{review_id}` | No | Implemented |
| `PATCH` | `/reviews/{review_id}` | Yes | Implemented |
| `DELETE` | `/reviews/{review_id}` | Yes | Implemented |
| `POST` | `/reviews/{review_id}/like` | Yes | Implemented |
| `POST` | `/reviews/{review_id}/dislike` | Yes | Implemented |
| `DELETE` | `/reviews/{review_id}/reaction` | Yes | Implemented |
| `GET` | `/movies/{movie_id}` | No | Implemented |
| `GET` | `/movies/{movie_id}/similar` | No | Implemented |
| `GET` | `/movies/{movie_id}/recommendations` | No | Implemented |
| `GET` | `/movies/search` | No | Implemented |
| `GET` | `/movies/trending` | No | Implemented |

Follower/following lists, a user's reviews, the following feed, and a movie's reviews
accept `limit` (1-100, default 20) and `offset` (default 0). Count endpoints return
`{"count": N}`. Follow and block actions return `204 No Content` on success.

`GET /users/search` is public and searches usernames case-insensitively. `search` is
required and surrounding whitespace is trimmed; `limit` (1-100, default 20) and `offset`
(default 0) paginate the results, which are ordered by username. It returns a JSON array
of public users (`id`, `username`, `profilePictureURL`, `country`), or `[]` when nothing
matches. The LIKE wildcards `%` and `_`, and backslashes, are treated literally, so they
cannot be used to broaden the search. It returns `400 Bad Request` for a blank `search`
or invalid pagination parameters.

Reviews are returned newest-updated first as a JSON array of review objects (`id`,
`user_id`, `movie_id`, `score`, `content`, `containsSpoilers`, `created_at`,
`updated_at`), with `id` as the tiebreaker. `GET /reviews/following` returns reviews
from every user the authenticated user follows. If a block exists in either direction,
`GET /users/username/{username}/reviews` returns `403 Forbidden`.
`GET /movies/{movie_id}/reviews` is public and returns `400 Bad Request` for movie IDs
that are not positive 32-bit integers.

`GET /movies/{movie_id}/reviews/average` is public and returns the average score as
`{"averageRating": N}`. It returns `0` when the movie exists but has no reviews, `400`
for movie IDs that are not positive 32-bit integers, and `404` when the movie is not in
the local cache.

`GET /movies/{movie_id}/reviews/count` is public and returns the total number of reviews
as `{"count": N}`. It returns `0` when the movie exists but has no reviews, `400` for
movie IDs that are not positive 32-bit integers, and `404` when the movie is not in the
local cache.

`POST /movies/{movie_id}/reviews` creates a review for the authenticated user. `score`
is required and must be between 1 and 10; `content` may be empty and is capped at
10,000 characters; `containsSpoilers` defaults to `false`. It returns `201 Created`
with the review, `400` for invalid input, and `409` if the user already reviewed that
movie. The movie must already exist in the local `movies` cache, so call
`GET /movies/{movie_id}` first if necessary; missing movies currently surface as `500`
because foreign-key violations are not mapped.

`GET /reviews/{review_id}` returns a single review and is public. It returns `404` if
the review does not exist, and `400` when the ID is not a positive 64-bit integer.

`PATCH /reviews/{review_id}` partially updates the authenticated user's review. Any
subset of `score`, `content`, and `containsSpoilers` may be sent, but at least one is
required; omitted fields keep their current values. A successful update returns `200 OK`
with the review and a refreshed `updated_at`. It returns `400` for invalid input or an
empty field set, and `404` if the review does not exist or belongs to another user.

`DELETE /reviews/{review_id}` deletes the authenticated user's review and returns
`204 No Content`. It returns `404` if the review does not exist or belongs to another
user. The review ID is validated on both routes and returns `400` when it is not a
positive 64-bit integer.

### Reactions

`POST /reviews/{review_id}/like` sets a `1` reaction and `POST /reviews/{review_id}/dislike`
sets `-1`. Both are upserts: reacting again updates the existing value, so liking and
then disliking flips the same row. They return `200 OK` with the reaction
(`target_id`, `user_id`, `value`, `created_at`), `400` for an invalid review ID, and
`404` if the review does not exist.

`DELETE /reviews/{review_id}/reaction` removes the caller's reaction and returns
`204 No Content`. It returns `404` when the review does not exist or the caller has no
reaction on it. All reaction routes require authentication.

### Movies

`GET /movies/{movie_id}` is public and returns the movie from the local cache. On a
cache miss it fetches the movie from TMDB (10-second timeout, cancelled if the client
disconnects), stores it in `movies`, and returns it. Responses use `tmdbId`, `title`,
`posterPath`, and `cachedAt`, where `cachedAt` is the timestamp at which the movie was
first cached.

It returns `400 Bad Request` for IDs that are not positive 32-bit integers, `404 Not
Found` if TMDB has no such movie, and `500 Internal Server Error` for other TMDB or
database failures, including a missing `TMDB_API_KEY`. The cache never expires.

`GET /movies/trending` is public and returns TMDB's daily trending movies. It accepts
`page` (1-500, default `1`) and returns `page`, `totalPages`, `totalResults`, and a
`results` array whose items use the same shape as movie lookups (`tmdbId`, `title`,
`posterPath`, `cachedAt`). Trending results are cached in `movies` on a best-effort
basis: cache write failures are logged but do not fail the request. It returns `400 Bad
Request` for invalid pages and `500 Internal Server Error` for TMDB failures (including a
missing `TMDB_API_KEY`).

`GET /movies/{movie_id}/similar` is public and returns TMDB's similar movies for a
movie. It accepts `page` (1-500, default `1`) and uses the same paginated response shape
as trending, caching results best-effort. It returns `400 Bad Request` for invalid movie
IDs or pages, `404 Not Found` if TMDB has no such movie, and `500 Internal Server Error`
for other TMDB failures (including a missing `TMDB_API_KEY`).

`GET /movies/{movie_id}/recommendations` is public and returns TMDB's recommended
movies for a movie. It accepts `page` (1-500, default `1`) and uses the same paginated
response shape as trending, caching results best-effort. It returns `400 Bad Request` for
invalid movie IDs or pages, `404 Not Found` if TMDB has no such movie, and `500 Internal
Server Error` for other TMDB failures (including a missing `TMDB_API_KEY`).

`GET /movies/search` is public and searches TMDB for movies. `search` is required and
surrounding whitespace is trimmed; `page` (1-500, default `1`) is optional. It uses the
same paginated response shape as trending, caching results best-effort. It returns `400
Bad Request` for a blank `search` or invalid page, and `500 Internal Server Error` for TMDB
failures (including a missing `TMDB_API_KEY`).

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
| `PATCH /reviews/{review_id}` | 64 KiB |

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

Search users by username:

```bash
curl -i "http://localhost:8080/api/v1/users/search?search=ad&limit=10&offset=0"
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

Get a movie's average score and review count:

```bash
curl -i "http://localhost:8080/api/v1/movies/550/reviews/average"
curl -i "http://localhost:8080/api/v1/movies/550/reviews/count"
```

Get a movie (served from the local cache, fetched from TMDB on a miss):

```bash
curl -i http://localhost:8080/api/v1/movies/550
```

List daily trending movies:

```bash
curl -i "http://localhost:8080/api/v1/movies/trending?page=1"
```

List movies similar to a given movie:

```bash
curl -i "http://localhost:8080/api/v1/movies/550/similar?page=1"
```

List movies recommended for a given movie:

```bash
curl -i "http://localhost:8080/api/v1/movies/550/recommendations?page=1"
```

Create a review (the movie must already be in the local cache):

```bash
curl -i -b cookies.txt -X POST http://localhost:8080/api/v1/movies/550/reviews \
  -H 'Content-Type: application/json' \
  -d '{"score":8,"content":"Great movie","containsSpoilers":false}'
```

Partially update that review (only the fields sent are changed; use the `id` returned by
the creation request):

```bash
curl -i -b cookies.txt -X PATCH http://localhost:8080/api/v1/reviews/1 \
  -H 'Content-Type: application/json' \
  -d '{"score":9,"containsSpoilers":true}'
```

Delete that review:

```bash
curl -i -b cookies.txt -X DELETE http://localhost:8080/api/v1/reviews/1
```

Like, dislike, and remove the reaction:

```bash
curl -i -b cookies.txt -X POST http://localhost:8080/api/v1/reviews/1/like
curl -i -b cookies.txt -X POST http://localhost:8080/api/v1/reviews/1/dislike
curl -i -b cookies.txt -X DELETE http://localhost:8080/api/v1/reviews/1/reaction
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

Unit tests cover services (including user search, review listing, creation, update,
deletion, average rating, review counts, reactions, and movie lookup, search, trending,
similar and recommended movies with a faked TMDB client), handlers (including user
search, movie status and page mapping, and JSON body limits), middleware, and repository
helpers (including LIKE-pattern escaping). They use fakes from
`internal/testutil`. Repository integration tests are not included because they require
a running PostgreSQL instance.

## Notes and limitations

- User search by username is implemented at `GET /users/search` with pagination and
  literal wildcard handling.
- Review listing (per user, following feed, and per movie), average rating, review
  counts, creation, reads by ID, partial updates, deletion, and like/dislike reactions
  are implemented.
- Movie lookup, search, daily trending, similar movies, and recommendations are
  implemented: `GET /movies/{movie_id}` reads the local `movies` cache and falls back to
  TMDB on a miss, while `GET /movies/search`, `GET /movies/trending`,
  `GET /movies/{movie_id}/similar`, and `GET /movies/{movie_id}/recommendations` paginate
  TMDB results and cache them best-effort. The cache never expires.
- Creating a review requires the movie to already exist in the local `movies` cache;
  call `GET /movies/{movie_id}` first to populate it.
- Tables for movies, comments, and reactions exist; reactions are implemented for
  reviews only, and comments have no code.
- Blocks prevent follows, remove existing follow relationships, and hide reviews in both
  directions; they are not yet used to hide profiles, follower/following lists, comments,
  or reactions.
- There is no lint configuration in the repository yet.

## TODO

- **Movie bookmarks:** let authenticated users bookmark movies and manage a personal
  saved-movies list. This needs a `bookmarks` table keyed by user and movie, plus
  endpoints to add, remove, and list bookmarks with pagination.
- **Google login:** add Google OAuth as an alternative sign-in/sign-up flow. It should
  link a Google account to an existing user (or create one), then issue the same
  cookie-based session as the current login endpoint.
- **Caching:** improve the local TMDB cache — currently entries never expire. Add a TTL
  or invalidation strategy, and consider caching other expensive or frequently requested
  reads so responses stay fresh without hitting TMDB or Postgres unnecessarily.
