-- Remove board system enhancements from tft_state table
-- This removes obstacle positions and other board-specific features

-- No schema changes to revert since we're using JSONB storage
-- The board configuration fields will simply be omitted from the JSON data