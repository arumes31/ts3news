ALTER TABLE abyss_runs
    ADD COLUMN IF NOT EXISTS loot_summary JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS loot_count INTEGER NOT NULL DEFAULT 0 CHECK (loot_count >= 0);

COMMENT ON COLUMN abyss_runs.loot_summary IS
    'Bounded display labels of loot sealed in the run when it ended; rendered through HTML escaping in run history.';
COMMENT ON COLUMN abyss_runs.loot_count IS
    'Total loot rows sealed when the run ended, including labels beyond the bounded summary.';
