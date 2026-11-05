-- +goose Up
CREATE TABLE IF NOT EXISTS user_identities (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(50)  NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email            CITEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE (provider, provider_user_id),
    UNIQUE (user_id, provider),
    CHECK (char_length(btrim(provider)) > 0),
    CHECK (char_length(btrim(provider_user_id)) > 0)
);

-- +goose Down
DROP TABLE IF EXISTS user_identities;
