ALTER TABLE abyss_runs
    ADD COLUMN IF NOT EXISTS duration_ms BIGINT NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    ADD COLUMN IF NOT EXISTS floors_cleared INTEGER NOT NULL DEFAULT 0 CHECK (floors_cleared >= 0);
