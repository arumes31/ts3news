ALTER TABLE abyss_social_profiles
    DROP COLUMN IF EXISTS preferred_echo_uid,
    DROP COLUMN IF EXISTS ghost_echo_opt_in;
