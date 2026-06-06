-- Add persistent health and regeneration stacks to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS current_hp INTEGER;
ALTER TABLE users ADD COLUMN IF NOT EXISTS regen_stacks INTEGER NOT NULL DEFAULT 0;

-- Optional: If we want pets to persist between cycles, we could add a pets table.
-- For now, let's keep pets inside the fight cycle or until they die.
