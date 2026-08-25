CREATE TABLE IF NOT EXISTS abyss_secret_boss_chains (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    stage INTEGER NOT NULL DEFAULT 0 CHECK (stage BETWEEN 0 AND 3),
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
