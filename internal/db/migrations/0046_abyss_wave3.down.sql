-- Drop tables
DROP TABLE IF EXISTS abyss_bestiary;
DROP TABLE IF EXISTS abyss_lore_unlocked;

-- Remove columns from abyss_active
ALTER TABLE abyss_active DROP COLUMN IF EXISTS floor_type;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS modifier;
ALTER TABLE abyss_active DROP COLUMN IF EXISTS event_state;
