CREATE TABLE IF NOT EXISTS abyss_pet_gifts (
    code TEXT PRIMARY KEY,
    pet_id BIGINT NOT NULL UNIQUE REFERENCES user_pets(pet_id) ON DELETE CASCADE,
    sender_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    recipient_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',
    CHECK (sender_uid <> recipient_uid)
);

CREATE INDEX IF NOT EXISTS idx_abyss_pet_gifts_recipient
    ON abyss_pet_gifts (recipient_uid, expires_at DESC);
