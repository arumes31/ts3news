-- User ultimate skill slot
ALTER TABLE users ADD COLUMN IF NOT EXISTS ultimate_skill_id TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS ultimate_cooldown INTEGER DEFAULT 0;

-- User ultimate skills collection
CREATE TABLE IF NOT EXISTS user_ultimate_skills (
    client_uid       TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    ultimate_id      TEXT NOT NULL,
    obtained         TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (client_uid, ultimate_id)
);

-- Add ultimate skills count to users table for quick lookup
ALTER TABLE users ADD COLUMN IF NOT EXISTS ultimate_skills_count INTEGER DEFAULT 0;
