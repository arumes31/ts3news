-- Remove scrap_stack column from users table
ALTER TABLE users DROP COLUMN IF EXISTS scrap_stack;
