CREATE INDEX IF NOT EXISTS abyss_boss_kills_player_best_idx
    ON abyss_boss_kills (client_uid, depth DESC, kill_time_ms, killed_at DESC);
