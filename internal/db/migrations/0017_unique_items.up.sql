-- User unique items collection
CREATE TABLE IF NOT EXISTS user_unique_items (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    item_name  TEXT NOT NULL,
    rarity     INTEGER NOT NULL,
    power      REAL NOT NULL,
    obtained   TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, item_name)
);

-- Add unique items count to users table for quick lookup
ALTER TABLE users ADD COLUMN IF NOT EXISTS unique_items_count INTEGER DEFAULT 0;
