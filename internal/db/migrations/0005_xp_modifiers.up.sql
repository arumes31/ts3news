-- State for the XP-modifier systems: streaks, daily login, sloth decay,
-- corrupted artifacts, and the cached client database id (for offline group ops).
ALTER TABLE users ADD COLUMN IF NOT EXISTS streak_days     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_poke_date  DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_date DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_decay_date DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS artifact_mult   DOUBLE PRECISION NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN IF NOT EXISTS artifact_name   TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS artifact_expires DATE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS cldbid          INTEGER NOT NULL DEFAULT 0;
