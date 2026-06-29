ALTER TABLE abyss_boss_kills DROP CONSTRAINT IF EXISTS abyss_boss_kills_tier_check;
ALTER TABLE abyss_boss_kills DROP COLUMN IF EXISTS tier;
