-- Per-user record of which game offers have already been sent. Dedup is keyed on
-- the stable TeamSpeak unique identifier (client_uid) plus a cross-source game
-- key (a normalised title), so the same game from different sources counts once.
CREATE TABLE IF NOT EXISTS sent_notifications (
    client_uid      TEXT NOT NULL,
    game_key        TEXT NOT NULL,
    game_title      TEXT,
    client_nickname TEXT,
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, game_key)
);

CREATE INDEX IF NOT EXISTS idx_sent_notifications_sent_at ON sent_notifications (sent_at);
