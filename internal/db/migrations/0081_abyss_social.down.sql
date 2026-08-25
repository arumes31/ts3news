DROP TABLE IF EXISTS abyss_weekly_boss_contributions;
DROP TABLE IF EXISTS abyss_weekly_bosses;
DROP TABLE IF EXISTS abyss_weekly_rivals;
DROP TABLE IF EXISTS abyss_social_notifications;
DROP TABLE IF EXISTS abyss_social_profiles;
DROP TABLE IF EXISTS abyss_helper_bonds;
DROP TABLE IF EXISTS abyss_deaths;
DROP TABLE IF EXISTS abyss_pet_memorials;
DROP INDEX IF EXISTS idx_user_pets_active_slot;
ALTER TABLE user_pets
    DROP COLUMN IF EXISTS autoskills,
    DROP COLUMN IF EXISTS training_count,
    DROP COLUMN IF EXISTS trained_on,
    DROP COLUMN IF EXISTS active_slot;
