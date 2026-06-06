-- User skills (5 slots)
CREATE TABLE IF NOT EXISTS user_skills (
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    slot       INTEGER NOT NULL, -- 1 to 5
    skill_id   TEXT NOT NULL,
    PRIMARY KEY (client_uid, slot)
);

-- Mob spell assignment logic is dynamic, but we can store if needed.
-- For now, let's keep it in-memory scaling based on mob level.
