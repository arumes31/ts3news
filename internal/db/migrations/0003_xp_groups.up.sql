-- Cache of auto-created level-tier server groups (XP_SERVER_GROUPS). Each tier maps
-- to a TeamSpeak server group id and the uploaded icon id. Rows are removed when a
-- tier group is deleted (e.g. when it becomes empty).
CREATE TABLE IF NOT EXISTS level_groups (
    tier       INTEGER PRIMARY KEY,
    sgid       INTEGER NOT NULL,
    icon_id    BIGINT  NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Track which tier group a user currently holds, so we can move them on tier-up
-- and detect when an old tier group is left empty.
ALTER TABLE users ADD COLUMN IF NOT EXISTS group_tier INTEGER NOT NULL DEFAULT 0;
