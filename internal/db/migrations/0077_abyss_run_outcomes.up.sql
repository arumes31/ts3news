ALTER TABLE abyss_runs
    ADD COLUMN IF NOT EXISTS end_reason TEXT NOT NULL DEFAULT 'legacy';

UPDATE abyss_runs
   SET end_reason = CASE WHEN victory THEN 'banked' ELSE 'defeat' END
 WHERE end_reason = 'legacy';

ALTER TABLE abyss_runs DROP CONSTRAINT IF EXISTS abyss_runs_end_reason_check;

ALTER TABLE abyss_runs
    ADD CONSTRAINT abyss_runs_end_reason_check
    CHECK (end_reason IN ('legacy', 'banked', 'defeat', 'revive_failed', 'conceded', 'timeout'));
