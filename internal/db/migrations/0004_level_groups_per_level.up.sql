-- XP server groups are now per individual level (e.g. "Peasant I", "Peasant II")
-- rather than per tier. Rename the columns accordingly.
ALTER TABLE level_groups RENAME COLUMN tier TO level;
ALTER TABLE users RENAME COLUMN group_tier TO group_level;
