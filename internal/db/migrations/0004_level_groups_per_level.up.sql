-- XP server groups are now per individual level (e.g. "Peasant I", "Peasant II")
-- rather than per tier. Rename the columns accordingly.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'level_groups' AND column_name = 'tier') AND
       NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'level_groups' AND column_name = 'level') THEN
        ALTER TABLE level_groups RENAME COLUMN tier TO level;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'group_tier') AND
       NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'group_level') THEN
        ALTER TABLE users RENAME COLUMN group_tier TO group_level;
    END IF;
END $$;
