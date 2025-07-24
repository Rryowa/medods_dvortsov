-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    guid UUID NOT NULL UNIQUE
);

CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    selector TEXT NOT NULL UNIQUE,
    verifier_hash TEXT NOT NULL,
    client_ip TEXT NOT NULL,
    user_agent TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'active',
    access_token_jti TEXT NOT NULL DEFAULT ''
);

CREATE INDEX ON sessions (selector);
CREATE INDEX ON sessions (user_id);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
