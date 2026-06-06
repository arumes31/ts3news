DROP TABLE IF EXISTS user_ultimate_skills;
ALTER TABLE users DROP COLUMN IF EXISTS ultimate_skill_id;
ALTER TABLE users DROP COLUMN IF EXISTS ultimate_cooldown;
ALTER TABLE users DROP COLUMN IF EXISTS ultimate_skills_count;
