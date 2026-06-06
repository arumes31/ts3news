-- Replace time-based expiry with durability
ALTER TABLE user_gear ADD COLUMN IF NOT EXISTS durability INTEGER NOT NULL DEFAULT 50;

-- Convert existing expiry timestamps to durability (1 day = 50 durability as a heuristic)
UPDATE user_gear SET durability = GREATEST(1, EXTRACT(EPOCH FROM (expires - NOW())) / 1728) -- 1728 = 86400 / 50
WHERE expires IS NOT NULL;

ALTER TABLE user_gear DROP COLUMN IF EXISTS expires;

ALTER TABLE users ADD COLUMN IF NOT EXISTS artifact_durability INTEGER NOT NULL DEFAULT 0;

-- Convert artifact expiry
UPDATE users SET artifact_durability = GREATEST(1, EXTRACT(EPOCH FROM (artifact_expires - NOW())) / 1728)
WHERE artifact_expires IS NOT NULL;

ALTER TABLE users DROP COLUMN IF EXISTS artifact_expires;

-- Titles still expire based on time (7 days), so no change there.
