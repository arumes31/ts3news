ALTER TABLE user_inventory
    ADD COLUMN IF NOT EXISTS locked BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_user_inventory_recent
    ON user_inventory (client_uid, acquired_at DESC);
