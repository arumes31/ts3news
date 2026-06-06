-- Track consecutive losses for auto-balancing
ALTER TABLE users ADD COLUMN IF NOT EXISTS consecutive_losses INTEGER NOT NULL DEFAULT 0;
