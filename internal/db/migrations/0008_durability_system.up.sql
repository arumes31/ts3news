-- Replace time-based expiry with durability
ALTER TABLE user_gear ADD COLUMN IF NOT EXISTS durability INTEGER NOT NULL DEFAULT 50;
ALTER TABLE user_gear DROP COLUMN IF EXISTS expires;

ALTER TABLE users ADD COLUMN IF NOT EXISTS artifact_durability INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users DROP COLUMN IF EXISTS artifact_expires;

-- Titles still expire based on time (7 days), so no change there.
