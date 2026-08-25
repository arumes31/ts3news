DROP INDEX IF EXISTS abyss_bestiary_recent_idx;
ALTER TABLE abyss_bestiary DROP COLUMN IF EXISTS last_kill_at;
