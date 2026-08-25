ALTER TABLE abyss_social_profiles
    ADD COLUMN IF NOT EXISTS ghost_echo_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS preferred_echo_uid TEXT REFERENCES users(client_uid) ON DELETE SET NULL;
