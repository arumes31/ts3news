ALTER TABLE users RENAME COLUMN group_level TO group_tier;
ALTER TABLE level_groups RENAME COLUMN level TO tier;
