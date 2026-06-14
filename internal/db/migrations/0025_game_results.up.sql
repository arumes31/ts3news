-- Per-play results for the arcade and auto-battler, powering the "most wins"
-- leaderboards (1-day / 30-day / all-time).
CREATE TABLE IF NOT EXISTS game_results (
    id         BIGSERIAL PRIMARY KEY,
    client_uid TEXT        NOT NULL,
    game       TEXT        NOT NULL, -- 'arcade' | 'tft'
    won        BOOLEAN     NOT NULL,
    net        BIGINT      NOT NULL DEFAULT 0, -- net gold change for the play
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Leaderboard queries filter by game + time window and group by player.
CREATE INDEX IF NOT EXISTS idx_game_results_lb ON game_results (game, created_at);
