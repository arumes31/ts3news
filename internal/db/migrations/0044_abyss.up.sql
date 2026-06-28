-- The Abyss: an endless push-your-luck PvE dungeon that drives the real combat
-- engine with the player's actual character (gear, skills, ultimate, pets,
-- consumables, artifact, title). Each cleared floor adds a bonus to an escrowed
-- cache that is paid out on Bank but forfeited on death.

-- One row per active descent. Server-authoritative depth + escrowed bonus gold so
-- the client can never lie about how deep it is or how much is at stake.
CREATE TABLE IF NOT EXISTS abyss_active (
    client_uid TEXT        PRIMARY KEY,
    depth      INTEGER     NOT NULL DEFAULT 0,
    escrow     BIGINT      NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One row per finished descent (banked or died), powering the "deepest descent"
-- leaderboards (1-day / 30-day / all-time).
CREATE TABLE IF NOT EXISTS abyss_runs (
    id          BIGSERIAL   PRIMARY KEY,
    client_uid  TEXT        NOT NULL,
    depth       INTEGER     NOT NULL,
    gold_banked BIGINT      NOT NULL DEFAULT 0,
    victory     BOOLEAN     NOT NULL DEFAULT FALSE, -- true = banked, false = died
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abyss_runs_lb ON abyss_runs (created_at, depth);

-- Personal best depth, surfaced on the Abyss page.
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_best_depth INTEGER NOT NULL DEFAULT 0;
