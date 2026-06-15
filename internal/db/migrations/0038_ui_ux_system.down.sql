-- UI/UX System Down Migration
-- Removes player settings storage

-- Drop index
DROP INDEX IF EXISTS idx_player_settings_client_uid;

-- Drop player settings table
DROP TABLE IF EXISTS player_settings;
