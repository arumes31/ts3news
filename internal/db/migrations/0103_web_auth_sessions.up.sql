-- Replace reusable plaintext URL credentials with one-time login grants and
-- independent revocable sessions. Only SHA-256 digests are persisted.
CREATE TABLE web_login_grants (
    token_hash BYTEA PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_web_login_grants_expiry ON web_login_grants (expires_at);

CREATE TABLE web_sessions (
    token_hash BYTEA PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_web_sessions_uid ON web_sessions (client_uid);
CREATE INDEX idx_web_sessions_expiry ON web_sessions (expires_at);

-- Revoke every legacy persistent bearer during deployment. The application no
-- longer reads this column; dropping it prevents accidental credential reuse.
DROP INDEX IF EXISTS idx_users_web_token;
ALTER TABLE users DROP COLUMN IF EXISTS web_token;
