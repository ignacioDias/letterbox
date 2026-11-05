-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username            CITEXT      NOT NULL UNIQUE,
    email               CITEXT      NOT NULL UNIQUE,
    password_hash       text        NOT NULL,
    profile_picture_url text,
    country             VARCHAR(2),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CHECK (expires_at > created_at)
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Local cache of TMDB data; TMDB remains the source of truth.
CREATE TABLE IF NOT EXISTS movies (
    tmdb_id      integer     PRIMARY KEY,
    title        text        NOT NULL,
    poster_path  text,
    cached_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS reviews (
    id                BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id           BIGINT      NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    movie_id          integer     NOT NULL REFERENCES movies(tmdb_id) ON DELETE RESTRICT,
    score             smallint    NOT NULL CHECK (score BETWEEN 1 AND 10),
    content           text        CHECK (content IS NULL OR length(content) <= 10000),
    contains_spoilers boolean     NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, movie_id)
);
CREATE INDEX IF NOT EXISTS reviews_user_updated_idx ON reviews (user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS reviews_movie_updated_idx ON reviews (movie_id, updated_at DESC, id DESC);

-- A NULL parent_comment_id is a review comment; the composite FK keeps replies on the same review.
CREATE TABLE IF NOT EXISTS comments (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    review_id         BIGINT      NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    parent_comment_id BIGINT,
    user_id           BIGINT      NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    content           text        NOT NULL CHECK (length(content) BETWEEN 1 AND 2000),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, review_id),
    FOREIGN KEY (parent_comment_id, review_id)
        REFERENCES comments (id, review_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS comments_review_created_idx ON comments (review_id, created_at);
CREATE INDEX IF NOT EXISTS comments_parent_idx ON comments (parent_comment_id)
    WHERE parent_comment_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS comments_user_idx ON comments (user_id);

CREATE TABLE IF NOT EXISTS review_reactions (
    user_id    BIGINT      NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    review_id  BIGINT      NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    value      smallint    NOT NULL CHECK (value IN (-1, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, review_id)
);
CREATE INDEX IF NOT EXISTS review_reactions_review_idx ON review_reactions (review_id);

CREATE TABLE IF NOT EXISTS comment_reactions (
    user_id    BIGINT      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    comment_id BIGINT      NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    value      smallint    NOT NULL CHECK (value IN (-1, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, comment_id)
);
CREATE INDEX IF NOT EXISTS comment_reactions_comment_idx ON comment_reactions (comment_id);

CREATE TABLE IF NOT EXISTS follows (
    follower_id BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_id BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followed_id),
    CHECK (follower_id <> followed_id)
);
CREATE INDEX IF NOT EXISTS follows_followed_idx ON follows (followed_id);

CREATE TABLE IF NOT EXISTS blocks (
    blocker_id BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);
CREATE INDEX IF NOT EXISTS blocks_blocked_idx ON blocks (blocked_id);

-- +goose Down
DROP TABLE IF EXISTS blocks;
DROP TABLE IF EXISTS follows;
DROP TABLE IF EXISTS comment_reactions;
DROP TABLE IF EXISTS review_reactions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS movies;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
