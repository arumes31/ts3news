-- Add board system enhancements to tft_state table
-- This includes obstacle positions and other board-specific features

-- We'll store board configuration in the JSONB data column
-- The board configuration will include:
-- - obstacle_positions: array of integers representing blocked positions
-- - grid_type: 'hex' or 'square' for different grid layouts
-- - board_size: dimensions of the board

-- No schema changes needed since we're using JSONB storage
-- Just documenting the new fields that will be stored in the data column
-- {
--   "obstacle_positions": [5, 12, 18],  -- example obstacle positions
--   "grid_type": "hex",
--   "board_size": {"width": 4, "height": 7}
-- }