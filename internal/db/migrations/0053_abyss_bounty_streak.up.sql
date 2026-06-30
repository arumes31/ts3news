-- Bounty streak: consecutive days the player has claimed the daily bounty. Drives
-- an escalating token bonus. Reset to 1 whenever a day is missed (handled in code).
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_bounty_streak INTEGER NOT NULL DEFAULT 0 CHECK (abyss_bounty_streak >= 0);
