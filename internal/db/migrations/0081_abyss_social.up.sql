-- Abyss social and companion tranche (AB-176..200).

ALTER TABLE user_pets
    ADD COLUMN IF NOT EXISTS active_slot SMALLINT NOT NULL DEFAULT 0 CHECK (active_slot BETWEEN 0 AND 2),
    ADD COLUMN IF NOT EXISTS trained_on DATE,
    ADD COLUMN IF NOT EXISTS training_count SMALLINT NOT NULL DEFAULT 0 CHECK (training_count BETWEEN 0 AND 3),
    ADD COLUMN IF NOT EXISTS autoskills JSONB NOT NULL DEFAULT '{"heal":true}'::jsonb;

-- Existing accounts keep their oldest living companion active in slot one.
WITH first_pet AS (
    SELECT DISTINCT ON (client_uid) pet_id
      FROM user_pets
     ORDER BY client_uid, captured_at, pet_id
)
UPDATE user_pets p
   SET active_slot=1
  FROM first_pet f
 WHERE p.pet_id=f.pet_id AND p.active_slot=0;

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_pets_active_slot
    ON user_pets (client_uid, active_slot) WHERE active_slot > 0;

CREATE TABLE IF NOT EXISTS abyss_pet_memorials (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    name TEXT NOT NULL,
    mob_type TEXT NOT NULL,
    level INTEGER NOT NULL CHECK (level >= 0),
    loyalty INTEGER NOT NULL CHECK (loyalty BETWEEN 0 AND 100),
    fallen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abyss_pet_memorials_owner
    ON abyss_pet_memorials (client_uid, fallen_at DESC);

CREATE TABLE IF NOT EXISTS abyss_deaths (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    depth INTEGER NOT NULL CHECK (depth >= 0),
    killer_name TEXT NOT NULL,
    killer_family TEXT NOT NULL,
    died_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abyss_deaths_owner
    ON abyss_deaths (client_uid, died_at DESC);

CREATE TABLE IF NOT EXISTS abyss_helper_bonds (
    uid_low TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    uid_high TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    assists INTEGER NOT NULL DEFAULT 0 CHECK (assists >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (uid_low, uid_high),
    CHECK (uid_low < uid_high)
);

CREATE TABLE IF NOT EXISTS abyss_social_profiles (
    client_uid TEXT PRIMARY KEY REFERENCES users(client_uid) ON DELETE CASCADE,
    bank_feed_opt_in BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS abyss_social_notifications (
    id BIGSERIAL PRIMARY KEY,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_abyss_social_notifications_owner
    ON abyss_social_notifications (client_uid, created_at DESC);

CREATE TABLE IF NOT EXISTS abyss_weekly_rivals (
    week_key TEXT NOT NULL,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    rival_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    target_depth INTEGER NOT NULL CHECK (target_depth > 0),
    claimed_at TIMESTAMPTZ,
    PRIMARY KEY (week_key, client_uid),
    CHECK (client_uid <> rival_uid)
);

CREATE TABLE IF NOT EXISTS abyss_weekly_bosses (
    week_key TEXT PRIMARY KEY,
    boss_name TEXT NOT NULL,
    max_hp BIGINT NOT NULL CHECK (max_hp > 0),
    current_hp BIGINT NOT NULL CHECK (current_hp >= 0),
    defeated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS abyss_weekly_boss_contributions (
    week_key TEXT NOT NULL REFERENCES abyss_weekly_bosses(week_key) ON DELETE CASCADE,
    client_uid TEXT NOT NULL REFERENCES users(client_uid) ON DELETE CASCADE,
    contribution_date DATE NOT NULL DEFAULT CURRENT_DATE,
    damage BIGINT NOT NULL CHECK (damage > 0),
    loot_label TEXT NOT NULL DEFAULT '',
    contributed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (week_key, client_uid, contribution_date)
);
