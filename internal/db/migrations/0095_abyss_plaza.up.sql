CREATE TABLE IF NOT EXISTS abyss_plaza_monuments (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    monument_key TEXT NOT NULL,
    gold_spent BIGINT NOT NULL CHECK (gold_spent > 0),
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, monument_key)
);

CREATE INDEX IF NOT EXISTS idx_abyss_plaza_monuments_public
    ON abyss_plaza_monuments (acquired_at DESC);
