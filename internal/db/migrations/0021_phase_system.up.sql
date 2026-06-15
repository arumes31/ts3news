-- Add phase system columns to tft_state table
ALTER TABLE tft_state 
ADD COLUMN phase VARCHAR(20) DEFAULT 'planning',
ADD COLUMN phase_timer INT DEFAULT 30,
ADD COLUMN round_number INT DEFAULT 1,
ADD COLUMN stage_number INT DEFAULT 1;