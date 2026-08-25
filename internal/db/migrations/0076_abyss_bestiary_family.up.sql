ALTER TABLE abyss_bestiary
    ADD COLUMN IF NOT EXISTS mob_family TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN abyss_bestiary.mob_family IS
    'Authoritative encounter family recorded from the defeated mob type; blank rows predate family tracking.';
