-- Synergy/Traits System Down Migration
-- Removes trait definitions and synergy tracking

-- Drop indexes
DROP INDEX IF EXISTS idx_trait_definitions_id;

-- Drop tables
DROP TABLE IF EXISTS trait_definitions;
