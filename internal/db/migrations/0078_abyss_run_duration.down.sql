ALTER TABLE abyss_runs
    DROP COLUMN IF EXISTS floors_cleared,
    DROP COLUMN IF EXISTS duration_ms;
