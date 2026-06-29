-- abyss_boss_kills was created without a tier column, but the boss-kill insert
-- and the topBossKills leaderboard query both reference k.tier. Add it so per-tier
-- speedrun boards work and the insert persists the difficulty.
ALTER TABLE abyss_boss_kills ADD COLUMN IF NOT EXISTS tier TEXT NOT NULL DEFAULT 'normal';
