CREATE TABLE IF NOT EXISTS abyss_affix_stats (
    client_uid  TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    affix_key   TEXT NOT NULL,
    runs        BIGINT NOT NULL DEFAULT 0 CHECK (runs >= 0),
    wins        BIGINT NOT NULL DEFAULT 0 CHECK (wins >= 0 AND wins <= runs),
    total_depth BIGINT NOT NULL DEFAULT 0 CHECK (total_depth >= 0),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, affix_key)
);
