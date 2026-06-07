-- Add scrap_stack column for tracking consecutive scrap drops (for XP stacking)
ALTER TABLE users ADD COLUMN IF NOT EXISTS scrap_stack INTEGER NOT NULL DEFAULT 0;
