-- Collapse back to the single-slot model: keep the oldest active ultimate as
-- the legacy equipped one before dropping the columns.
UPDATE users u
SET ultimate_skill_id = pick.ultimate_id, ultimate_cooldown = pick.current_cooldown
FROM (
    SELECT DISTINCT ON (client_uid) client_uid, ultimate_id, current_cooldown
    FROM user_ultimate_skills WHERE active
    ORDER BY client_uid, obtained, ultimate_id
) pick
WHERE u.client_uid = pick.client_uid;

ALTER TABLE user_ultimate_skills DROP COLUMN IF EXISTS active;
ALTER TABLE user_ultimate_skills DROP COLUMN IF EXISTS current_cooldown;
