-- Add phase system columns to tft_state table
ALTER TABLE tft_state 
ADD COLUMN phase TEXT NOT NULL DEFAULT 'planning',
ADD COLUMN phase_timer INTEGER NOT NULL DEFAULT 30,
ADD COLUMN round_number INTEGER NOT NULL DEFAULT 1,
ADD COLUMN stage_number INTEGER NOT NULL DEFAULT 1;