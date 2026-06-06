-- Per-user XP / leveling state, keyed on the TeamSpeak unique identifier.
-- last_seen drives the dead-user cleanup (users not seen for 6 months are purged).
CREATE TABLE IF NOT EXISTS users (
    client_uid TEXT PRIMARY KEY,
    nickname   TEXT,
    xp         INTEGER NOT NULL DEFAULT 0,
    level      INTEGER NOT NULL DEFAULT 1,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_last_seen ON users (last_seen);
