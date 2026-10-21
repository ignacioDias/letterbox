package database

// Schema objects are split into individual constants so database.go can execute them separately.
const (
	CreateCitextExtension = `
CREATE EXTENSION IF NOT EXISTS citext;
`

	CreateSetUpdatedAtFunction = `
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$;
`

	CreateUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username            citext      NOT NULL UNIQUE
                        CHECK (username ~ '^[A-Za-z0-9_]{3,30}$'),
    email               citext      NOT NULL UNIQUE
                        CHECK (length(email) <= 254),
    password_hash       text        NOT NULL,
    profile_picture_url text,
    country             char(2)
                        CHECK (country ~ '^[A-Z]{2}$'),  -- ISO 3166-1 alpha-2
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
`

	CreateUsersUpdatedAtTrigger = `
CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
`

	// Session cookie holds a random token; only its SHA-256 hash is stored here.
	CreateSessionsTable = `
CREATE TABLE IF NOT EXISTS sessions (
    id           bytea       PRIMARY KEY CHECK (octet_length(id) = 32),
    user_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    ip           inet,
    user_agent   text,
    CHECK (expires_at > created_at)
);
`

	CreateSessionsUserIDIndex = `
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
`

	CreateSessionsExpiresAtIndex = `
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
`

	// Local cache of TMDB data; TMDB remains the source of truth.
	CreateMoviesTable = `
CREATE TABLE IF NOT EXISTS movies (
    tmdb_id      integer     PRIMARY KEY,
    title        text        NOT NULL,
    poster_path  text,
    release_year smallint,
    cached_at    timestamptz NOT NULL DEFAULT now()
);
`

	CreateReviewsTable = `
CREATE TABLE IF NOT EXISTS reviews (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id           bigint      NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    movie_id          integer     NOT NULL REFERENCES movies(tmdb_id) ON DELETE RESTRICT,
    score             smallint    NOT NULL CHECK (score BETWEEN 1 AND 10),
    content           text        CHECK (content IS NULL OR length(content) BETWEEN 1 AND 10000),
    contains_spoilers boolean     NOT NULL DEFAULT false,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, movie_id)
);
`

	CreateReviewsUserCreatedIndex = `
CREATE INDEX reviews_user_created_idx ON reviews (user_id, created_at DESC);
`

	CreateReviewsMovieCreatedIndex = `
CREATE INDEX reviews_movie_created_idx ON reviews (movie_id, created_at DESC);
`

	CreateReviewsUpdatedAtTrigger = `
CREATE TRIGGER reviews_set_updated_at
    BEFORE UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
`

	// A NULL parent_comment_id is a review comment; the composite FK keeps replies on the same review.
	CreateCommentsTable = `
CREATE TABLE IF NOT EXISTS comments (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    review_id         bigint      NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    parent_comment_id bigint,
    user_id           bigint      NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    content           text        NOT NULL CHECK (length(content) BETWEEN 1 AND 2000),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (id, review_id),
    FOREIGN KEY (parent_comment_id, review_id)
        REFERENCES comments (id, review_id) ON DELETE CASCADE
);
`

	CreateCommentsReviewCreatedIndex = `
CREATE INDEX comments_review_created_idx ON comments (review_id, created_at);
`

	CreateCommentsParentIndex = `
CREATE INDEX comments_parent_idx ON comments (parent_comment_id)
    WHERE parent_comment_id IS NOT NULL;
`

	CreateCommentsUserIndex = `
CREATE INDEX comments_user_idx ON comments (user_id);
`

	CreateCommentsUpdatedAtTrigger = `
CREATE TRIGGER comments_set_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
`

	CreateReviewReactionsTable = `
CREATE TABLE IF NOT EXISTS review_reactions (
    user_id    bigint      NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    review_id  bigint      NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    value      smallint    NOT NULL CHECK (value IN (-1, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, review_id)
);
`

	CreateReviewReactionsReviewIndex = `
CREATE INDEX review_reactions_review_idx ON review_reactions (review_id);
`

	CreateCommentReactionsTable = `
CREATE TABLE IF NOT EXISTS comment_reactions (
    user_id    bigint      NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    comment_id bigint      NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    value      smallint    NOT NULL CHECK (value IN (-1, 1)),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, comment_id)
);
`

	CreateCommentReactionsCommentIndex = `
CREATE INDEX comment_reactions_comment_idx ON comment_reactions (comment_id);
`

	CreateFollowsTable = `
CREATE TABLE IF NOT EXISTS follows (
    follower_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (follower_id, followed_id),
    CHECK (follower_id <> followed_id)
);
`

	CreateFollowsFollowedIndex = `
CREATE INDEX follows_followed_idx ON follows (followed_id);
`

	CreateBlocksTable = `
CREATE TABLE IF NOT EXISTS blocks (
    blocker_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);
`

	CreateBlocksBlockedIndex = `
CREATE INDEX blocks_blocked_idx ON blocks (blocked_id);
`

	CreateRemoveFollowsOnBlockFunction = `
CREATE FUNCTION remove_follows_on_block() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    DELETE FROM follows
    WHERE (follower_id = NEW.blocker_id AND followed_id = NEW.blocked_id)
       OR (follower_id = NEW.blocked_id AND followed_id = NEW.blocker_id);
    RETURN NEW;
END;
$$;
`

	CreateBlocksRemoveFollowsTrigger = `
CREATE TRIGGER blocks_remove_follows
    AFTER INSERT ON blocks
    FOR EACH ROW EXECUTE FUNCTION remove_follows_on_block();
`
)

var SchemaStatements = []string{
	CreateCitextExtension,
	CreateSetUpdatedAtFunction,

	CreateUsersTable,
	CreateUsersUpdatedAtTrigger,

	CreateSessionsTable,
	CreateSessionsUserIDIndex,
	CreateSessionsExpiresAtIndex,

	CreateMoviesTable,

	CreateReviewsTable,
	CreateReviewsUserCreatedIndex,
	CreateReviewsMovieCreatedIndex,
	CreateReviewsUpdatedAtTrigger,

	CreateCommentsTable,
	CreateCommentsReviewCreatedIndex,
	CreateCommentsParentIndex,
	CreateCommentsUserIndex,
	CreateCommentsUpdatedAtTrigger,

	CreateReviewReactionsTable,
	CreateReviewReactionsReviewIndex,

	CreateCommentReactionsTable,
	CreateCommentReactionsCommentIndex,

	CreateFollowsTable,
	CreateFollowsFollowedIndex,

	CreateBlocksTable,
	CreateBlocksBlockedIndex,
	CreateRemoveFollowsOnBlockFunction,
	CreateBlocksRemoveFollowsTrigger,
}
