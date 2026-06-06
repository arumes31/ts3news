-- Drop triggers and functions for counter synchronization

DROP TRIGGER IF EXISTS unique_items_count_trigger ON user_unique_items;
DROP TRIGGER IF EXISTS ultimate_skills_count_trigger ON user_ultimate_skills;

DROP FUNCTION IF EXISTS sync_unique_items_count();
DROP FUNCTION IF EXISTS sync_ultimate_skills_count();