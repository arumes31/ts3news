-- Pacts: optional self-imposed challenge modifiers chosen at run start. Stored as a
-- space-separated set of validated pact keys on the active run, so they persist for
-- the whole descent and drive both the combat folding and the reward multiplier.
ALTER TABLE abyss_active ADD COLUMN IF NOT EXISTS pacts TEXT NOT NULL DEFAULT '';
