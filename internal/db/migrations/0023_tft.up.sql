-- Teamfight-style auto-battler board state (bench, board, shop) per user.
CREATE TABLE IF NOT EXISTS tft_state (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    data       JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
