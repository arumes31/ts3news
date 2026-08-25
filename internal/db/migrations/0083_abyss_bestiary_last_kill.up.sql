ALTER TABLE abyss_bestiary
    ADD COLUMN IF NOT EXISTS last_kill_at TIMESTAMPTZ;

UPDATE abyss_bestiary
SET last_kill_at = first_kill_at
WHERE last_kill_at IS NULL;

ALTER TABLE abyss_bestiary
    ALTER COLUMN last_kill_at SET DEFAULT NOW(),
    ALTER COLUMN last_kill_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS abyss_bestiary_recent_idx
    ON abyss_bestiary (client_uid, last_kill_at DESC);
