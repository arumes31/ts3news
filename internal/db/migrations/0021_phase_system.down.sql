-- Remove phase system columns from tft_state table
ALTER TABLE tft_state 
DROP COLUMN phase,
DROP COLUMN phase_timer,
DROP COLUMN round_number,
DROP COLUMN stage_number;