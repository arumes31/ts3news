-- Consecutive-floor-win streak within a single Abyss run. Grants a small stacking
-- combat buff (see abyssStreakBuff); resets on Downed or on a rested floor.
ALTER TABLE users ADD COLUMN IF NOT EXISTS abyss_win_streak INTEGER NOT NULL DEFAULT 0 CHECK (abyss_win_streak >= 0);
