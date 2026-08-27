CREATE TABLE IF NOT EXISTS abyss_api_tokens (
    client_uid   TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    token_prefix TEXT NOT NULL UNIQUE,
    token_hash   BYTEA NOT NULL UNIQUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
