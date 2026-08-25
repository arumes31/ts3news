DROP INDEX IF EXISTS idx_user_inventory_recent;

ALTER TABLE user_inventory
    DROP COLUMN IF EXISTS locked;
