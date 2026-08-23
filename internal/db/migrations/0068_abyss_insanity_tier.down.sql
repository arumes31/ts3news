ALTER TABLE abyss_active DROP CONSTRAINT IF EXISTS abyss_active_tier_check;
ALTER TABLE abyss_active ADD CONSTRAINT abyss_active_tier_check CHECK (tier IN ('normal','nightmare','hell'));

ALTER TABLE abyss_runs DROP CONSTRAINT IF EXISTS abyss_runs_tier_check;
ALTER TABLE abyss_runs ADD CONSTRAINT abyss_runs_tier_check CHECK (tier IN ('normal','nightmare','hell'));

ALTER TABLE abyss_boss_kills DROP CONSTRAINT IF EXISTS abyss_boss_kills_tier_check;
ALTER TABLE abyss_boss_kills ADD CONSTRAINT abyss_boss_kills_tier_check CHECK (tier IN ('normal','nightmare','hell'));
