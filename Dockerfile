FROM golang:1.27-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/letterbox ./cmd/letterbox

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system letterbox \
    && useradd --system --gid letterbox --home-dir /app letterbox

WORKDIR /app

COPY --from=build /out/letterbox /app/letterbox
COPY --chown=letterbox:letterbox assets /app/assets

USER letterbox

EXPOSE 8080

CMD ["/app/letterbox"]
