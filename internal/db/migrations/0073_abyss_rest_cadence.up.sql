ALTER TABLE abyss_active
    ADD COLUMN IF NOT EXISTS last_rest_depth INTEGER NOT NULL DEFAULT 0
    CHECK (last_rest_depth >= 0);
