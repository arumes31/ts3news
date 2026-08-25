ALTER TABLE users
    ADD COLUMN IF NOT EXISTS abyss_boss_tokens BIGINT NOT NULL DEFAULT 0
    CHECK (abyss_boss_tokens >= 0);
