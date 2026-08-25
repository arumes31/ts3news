ALTER TABLE abyss_runs ADD COLUMN IF NOT EXISTS hardcore BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_abyss_runs_hardcore_leaderboard
    ON abyss_runs (tier, depth DESC, gold_banked DESC, created_at DESC)
    WHERE hardcore = TRUE;
