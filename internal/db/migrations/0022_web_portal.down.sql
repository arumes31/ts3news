DROP TABLE IF EXISTS battle_history;
DROP TABLE IF EXISTS user_inventory;
DROP INDEX IF EXISTS idx_users_web_token;
ALTER TABLE users DROP COLUMN IF EXISTS web_token;
