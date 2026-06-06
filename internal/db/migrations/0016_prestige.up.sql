-- Prestige: when a user hits the level cap they prestige (level resets to 1,
-- prestige increments) and gain permanent stat bonuses + a prestige rank group.
ALTER TABLE users ADD COLUMN IF NOT EXISTS prestige INTEGER NOT NULL DEFAULT 0;
