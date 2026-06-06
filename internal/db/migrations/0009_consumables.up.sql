-- Consumables storage
CREATE TABLE IF NOT EXISTS user_consumables (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    cons_id    TEXT NOT NULL,
    remaining_fights INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (client_uid, cons_id)
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS current_hp_bonus INTEGER NOT NULL DEFAULT 0;
