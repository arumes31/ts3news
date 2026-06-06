-- Add class to users for skill specialization
ALTER TABLE users ADD COLUMN IF NOT EXISTS class TEXT DEFAULT 'Adventurer';

-- Create table for tracking user quests/achievements
CREATE TABLE IF NOT EXISTS user_quests (
    client_uid   TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    quest_type   TEXT NOT NULL, -- e.g., 'mobs_killed', 'bosses_killed', 'total_xp'
    progress     INTEGER NOT NULL DEFAULT 0,
    milestone    INTEGER NOT NULL DEFAULT 10, -- target for next reward
    total_earned INTEGER NOT NULL DEFAULT 0, -- lifetime count
    PRIMARY KEY (client_uid, quest_type)
);
