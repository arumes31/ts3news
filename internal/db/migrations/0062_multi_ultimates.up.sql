-- Players can run up to 3 distinct ultimates at once. Activation state and
-- per-ultimate cooldown move from users.ultimate_skill_id/ultimate_cooldown
-- (single slot, kept but no longer read) onto the collection table.
ALTER TABLE user_ultimate_skills ADD COLUMN IF NOT EXISTS active BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE user_ultimate_skills ADD COLUMN IF NOT EXISTS current_cooldown INTEGER NOT NULL DEFAULT 0;

-- Backfill: make sure the legacy equipped ultimate exists in the collection…
INSERT INTO user_ultimate_skills (client_uid, ultimate_id)
SELECT client_uid, ultimate_skill_id FROM users
WHERE ultimate_skill_id IS NOT NULL AND ultimate_skill_id <> ''
ON CONFLICT (client_uid, ultimate_id) DO NOTHING;

-- …and carries its live cooldown over as the active one.
UPDATE user_ultimate_skills us
SET active = TRUE, current_cooldown = COALESCE(u.ultimate_cooldown, 0)
FROM users u
WHERE u.client_uid = us.client_uid AND u.ultimate_skill_id = us.ultimate_id;

-- Auto-activate up to 3 owned ultimates per player (legacy equipped first,
-- then oldest finds) so existing collectors benefit immediately.
WITH ranked AS (
    SELECT client_uid, ultimate_id,
           ROW_NUMBER() OVER (PARTITION BY client_uid ORDER BY active DESC, obtained, ultimate_id) AS rn
    FROM user_ultimate_skills
)
UPDATE user_ultimate_skills us
SET active = TRUE
FROM ranked r
WHERE us.client_uid = r.client_uid AND us.ultimate_id = r.ultimate_id AND r.rn <= 3;
