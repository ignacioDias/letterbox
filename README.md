# Letterbox

A Letterbox copy: register, follow users, rate movies (1–10), and get a feed with
everything the people you follow are doing. You can like, dislike, and comment on
posts, message other users, and block users.

Messaging rules: if you don't follow a user you can send only one message and then
have to wait for them to allow the chat.

- **Backend:** Go (`net/http`)
- **Database:** PostgreSQL
- **Deployment:** Docker (planned)

## Status

Early scaffold. The HTTP server, routing entry point, and graceful shutdown are in
place; domain routes, persistence, and the Docker setup are not implemented yet.

## Project structure

```
cmd/letterbox/       Application entry point
internal/router/     Route registration (http.ServeMux)
internal/server/     HTTP server setup and graceful shutdown
internal/database/   Database connection and schema (in progress)
```

## Requirements

- Go 1.27+
- PostgreSQL (once persistence lands)

## Getting started

```sh
go run ./cmd/letterbox
```

The server listens on port `8080` and shuts down gracefully on `SIGINT`/`SIGTERM`
with a 30-second timeout.

## Configuration

Environment variables are loaded from `.env` (ignored by git; do not commit secrets).

| Variable | Description | Default |
| --- | --- | --- |
| `PORT` | HTTP listen port (currently hard-coded) | `8080` |
| `DATABASE_URL` | PostgreSQL connection string (planned) | — |

## Planned features

- [ ] User registration and authentication
- [ ] Follow / unfollow and blocking
- [ ] Movie ratings (1–10)
- [ ] Activity feed of followed users
- [ ] Likes, dislikes, and comments on posts
- [ ] Direct messaging with follow-gated chat requests
- [ ] PostgreSQL schema and migrations
- [ ] Docker / docker-compose setup

## Development

```sh
go build ./...    # compile
go test ./...     # run tests
go vet ./...      # static checks
```
