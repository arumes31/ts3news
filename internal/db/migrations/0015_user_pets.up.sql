-- Create table for tracking captured pets persistently
CREATE TABLE IF NOT EXISTS user_pets (
    pet_id       SERIAL PRIMARY KEY,
    client_uid   TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    mob_type     TEXT NOT NULL,
    level        INTEGER NOT NULL,
    hp           INTEGER NOT NULL,
    max_hp       INTEGER NOT NULL,
    str          INTEGER NOT NULL,
    def          INTEGER NOT NULL,
    spd          INTEGER NOT NULL,
    captured_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for faster lookup by user
CREATE INDEX IF NOT EXISTS idx_user_pets_uid ON user_pets(client_uid);
