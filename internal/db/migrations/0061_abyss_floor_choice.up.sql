-- Uncommitted candidate floor options offered by /api/abyss/descend before the
-- player picks one via /api/abyss/choose_floor. NULL when no choice is pending.
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS pending_floor_choice JSONB;
