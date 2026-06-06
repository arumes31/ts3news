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

-- Migrate legacy gear data safely if columns exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'weapon_id') THEN
        INSERT INTO user_gear (client_uid, slot, gear_id, expires)
        SELECT client_uid, 'weapon', weapon_id, weapon_expires FROM users WHERE weapon_id IS NOT NULL
        ON CONFLICT (client_uid, slot) DO NOTHING;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'armor_id') THEN
        INSERT INTO user_gear (client_uid, slot, gear_id, expires)
        SELECT client_uid, 'armor', armor_id, armor_expires FROM users WHERE armor_id IS NOT NULL
        ON CONFLICT (client_uid, slot) DO NOTHING;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'relic_id') THEN
        INSERT INTO user_gear (client_uid, slot, gear_id, expires)
        SELECT client_uid, 'relic', relic_id, relic_expires FROM users WHERE relic_id IS NOT NULL
        ON CONFLICT (client_uid, slot) DO NOTHING;
    END IF;
END $$;

-- Clean up old individual columns from previous iteration
ALTER TABLE users DROP COLUMN IF EXISTS weapon_id;
ALTER TABLE users DROP COLUMN IF EXISTS weapon_expires;
ALTER TABLE users DROP COLUMN IF EXISTS armor_id;
ALTER TABLE users DROP COLUMN IF EXISTS armor_expires;
ALTER TABLE users DROP COLUMN IF EXISTS relic_id;
ALTER TABLE users DROP COLUMN IF EXISTS relic_expires;
