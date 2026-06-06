-- Add total connection time tracking
ALTER TABLE users ADD COLUMN IF NOT EXISTS total_connection_seconds BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_session_connected_ms BIGINT NOT NULL DEFAULT 0;
