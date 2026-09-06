CREATE INDEX IF NOT EXISTS idx_abyss_economy_events_history
    ON abyss_economy_events (client_uid, id DESC);
