-- 0027_arcade3d_scores.up.sql

-- Create arcade3d_scores table for tracking individual 3D arcade game scores
-- Enables per-game leaderboards for all 31 arcade 3D games
CREATE TABLE IF NOT EXISTS arcade3d_scores (
    id              BIGSERIAL PRIMARY KEY,
    client_uid      TEXT        NOT NULL,
    game            TEXT        NOT NULL, -- e.g., 'snake', 'pong', 'tetris', etc.
    score           INTEGER     NOT NULL, -- the actual score achieved
    gold_awarded    INTEGER     NOT NULL DEFAULT 0, -- gold given as reward
    gear_won        TEXT, -- optional gear item won
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for efficient per-game leaderboard queries (game, score DESC)
CREATE INDEX IF NOT EXISTS idx_arcade3d_scores_game_score ON arcade3d_scores (game, score DESC);

-- Index for time-based filtering (recent scores, daily/weekly leaderboards)
CREATE INDEX IF NOT EXISTS idx_arcade3d_scores_created_at ON arcade3d_scores (created_at);

-- Index for user-specific stats (player's history per game)
CREATE INDEX IF NOT EXISTS idx_arcade3d_scores_client_game ON arcade3d_scores (client_uid, game);
