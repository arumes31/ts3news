ALTER TABLE abyss_runs
    DROP CONSTRAINT IF EXISTS abyss_runs_end_reason_check;

ALTER TABLE abyss_runs
    DROP COLUMN IF EXISTS end_reason;
