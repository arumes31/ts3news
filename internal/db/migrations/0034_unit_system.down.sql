-- Unit System Down Migration
-- Removes enhanced unit stats, abilities, and upgrades

-- Drop indexes
DROP INDEX IF EXISTS idx_unit_abilities_unit_key;
DROP INDEX IF EXISTS idx_unit_upgrades_unit_key;

-- Drop tables
DROP TABLE IF EXISTS unit_abilities;
DROP TABLE IF EXISTS unit_upgrades;
