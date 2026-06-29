DROP TABLE IF EXISTS abyss_boss_kills;
ALTER TABLE users DROP COLUMN IF EXISTS abyss_prestige;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS coop_uid;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS last_action_at;
