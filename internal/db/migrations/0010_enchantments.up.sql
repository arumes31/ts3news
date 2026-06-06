-- Add enchantment support to gear
ALTER TABLE user_gear ADD COLUMN IF NOT EXISTS enchantment_id TEXT;
