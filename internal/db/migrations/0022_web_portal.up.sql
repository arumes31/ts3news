-- Web portal: persistent per-user login token and a general item inventory.

-- Unique, persistent login token PM'd to each user every cycle (used in the
-- shortened login URL). NULL until first generated.
ALTER TABLE users ADD COLUMN IF NOT EXISTS web_token TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_web_token ON users (web_token) WHERE web_token IS NOT NULL;

-- Inventory of owned-but-unequipped gear (from shop purchases, battler loot the
-- player chose not to equip, etc.). Equipped gear continues to live in user_gear.
CREATE TABLE IF NOT EXISTS user_inventory (
    id          BIGSERIAL PRIMARY KEY,
    client_uid  TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    gear_id     TEXT NOT NULL,
    durability  INTEGER NOT NULL DEFAULT 50,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_inventory_uid ON user_inventory (client_uid);

-- History of web auto-battler fights for the per-user battle log.
CREATE TABLE IF NOT EXISTS battle_history (
    id         BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    mob_name   TEXT NOT NULL,
    victory    BOOLEAN NOT NULL,
    gold_won   BIGINT NOT NULL DEFAULT 0,
    gear_won   TEXT,
    fought_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_battle_history_uid ON battle_history (client_uid, fought_at DESC);
