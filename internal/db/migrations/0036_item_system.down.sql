-- Item System Down Migration
-- Removes item components, crafted items, recipes, and player inventory

-- Drop indexes
DROP INDEX IF EXISTS idx_player_items_client_uid;
DROP INDEX IF EXISTS idx_player_items_equipped_to;
DROP INDEX IF EXISTS idx_crafting_recipes_result_id;

-- Drop tables
DROP TABLE IF EXISTS player_items;
DROP TABLE IF EXISTS crafting_recipes;
DROP TABLE IF EXISTS crafted_items;
DROP TABLE IF EXISTS item_components;
