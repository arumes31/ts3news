-- Add titles to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS title_expires TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS title_mult FLOAT;

-- Create dedicated gear table for the 24 slots
CREATE TABLE IF NOT EXISTS user_gear (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    slot       TEXT NOT NULL,
    gear_id    TEXT NOT NULL,
    expires    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (client_uid, slot)
);

-- Clean up old individual columns from previous iteration
ALTER TABLE users DROP COLUMN IF EXISTS weapon_id;
ALTER TABLE users DROP COLUMN IF EXISTS weapon_expires;
ALTER TABLE users DROP COLUMN IF EXISTS armor_id;
ALTER TABLE users DROP COLUMN IF EXISTS armor_expires;
ALTER TABLE users DROP COLUMN IF EXISTS relic_id;
ALTER TABLE users DROP COLUMN IF EXISTS relic_expires;
