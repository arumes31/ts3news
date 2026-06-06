DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'group_level') AND
       NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'group_tier') THEN
        ALTER TABLE users RENAME COLUMN group_level TO group_tier;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'level_groups' AND column_name = 'level') AND
       NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'level_groups' AND column_name = 'tier') THEN
        ALTER TABLE level_groups RENAME COLUMN level TO tier;
    END IF;
END $$;
