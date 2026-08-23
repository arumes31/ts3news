-- The Insanity tier (x10 rewards / x20 danger) was added to the game tier
-- catalog, but the tier CHECK constraints still only allowed
-- normal/nightmare/hell, so entering an Insanity run failed with a constraint
-- violation ("couldn't save"). Widen all three to include 'insanity'.
ALTER TABLE abyss_active DROP CONSTRAINT IF EXISTS abyss_active_tier_check;
ALTER TABLE abyss_active ADD CONSTRAINT abyss_active_tier_check CHECK (tier IN ('normal','nightmare','hell','insanity'));

ALTER TABLE abyss_runs DROP CONSTRAINT IF EXISTS abyss_runs_tier_check;
ALTER TABLE abyss_runs ADD CONSTRAINT abyss_runs_tier_check CHECK (tier IN ('normal','nightmare','hell','insanity'));

ALTER TABLE abyss_boss_kills DROP CONSTRAINT IF EXISTS abyss_boss_kills_tier_check;
ALTER TABLE abyss_boss_kills ADD CONSTRAINT abyss_boss_kills_tier_check CHECK (tier IN ('normal','nightmare','hell','insanity'));
