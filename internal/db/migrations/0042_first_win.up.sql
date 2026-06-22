-- First win of the day tracking
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_win TIMESTAMPTZ;
