CREATE INDEX IF NOT EXISTS abyss_boss_kills_tier_boss_player_time_idx
    ON abyss_boss_kills (tier, boss_name, client_uid, kill_time_ms, depth DESC, killed_at);
