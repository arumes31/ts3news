ALTER TABLE users ADD COLUMN IF NOT EXISTS web_token TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_web_token
    ON users (web_token) WHERE web_token IS NOT NULL;

DROP TABLE IF EXISTS web_sessions;
DROP TABLE IF EXISTS web_login_grants;
