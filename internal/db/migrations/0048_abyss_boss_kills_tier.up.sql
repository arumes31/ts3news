-- abyss_boss_kills was created without a tier column, but the boss-kill insert
-- and the topBossKills leaderboard query both reference k.tier. Add it so per-tier
-- speedrun boards work and the insert persists the difficulty.
ALTER TABLE abyss_boss_kills ADD COLUMN IF NOT EXISTS tier TEXT NOT NULL DEFAULT 'normal';
-- Constrain to the supported tier set so leaderboards can trust the value.
ALTER TABLE abyss_boss_kills DROP CONSTRAINT IF EXISTS abyss_boss_kills_tier_check;
ALTER TABLE abyss_boss_kills ADD CONSTRAINT abyss_boss_kills_tier_check CHECK (tier IN ('normal','nightmare','hell'));
